package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Downloader 下载器结构体
type Downloader struct {
	URL          string        // 下载链接
	Output       string        // 输出文件名
	Threads      int           // 并发线程数
	FileSize     int64         // 文件总大小
	TempDir      string        // 临时文件目录
	TempFiles    []string      // 临时文件列表（按索引顺序）
	tempFilesMu  sync.Mutex    // 临时文件列表锁
	progress     uint64        // 已下载字节数（原子操作）
	progressMu   sync.Mutex    // 进度显示锁
	stopProgress chan struct{} // 停止进度显示的通道
	client       *http.Client  // HTTP客户端
	retryCount   int           // 重试次数
	timeout      time.Duration // 请求超时时间
	errors       []error       // 错误列表
	errorsMu     sync.Mutex    // 错误列表锁
	ctx          context.Context    // 上下文
	cancel       context.CancelFunc // 取消函数
	startTime    time.Time     // 开始时间
}

// NewDownloader 创建下载器实例
func NewDownloader(url, output string, threads int) (*Downloader, error) {
	// 验证URL
	if url == "" {
		return nil, errors.New("URL不能为空")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, errors.New("URL必须以http://或https://开头")
	}

	// 验证输出路径
	if output == "" {
		return nil, errors.New("输出文件名不能为空")
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(output)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("创建输出目录失败: %v", err)
		}
	}

	// 检查输出文件是否已存在
	if _, err := os.Stat(output); err == nil {
		fmt.Printf("警告：输出文件 %s 已存在，将被覆盖\n", output)
	}

	// 创建临时目录（基于输出文件的目录）
	tempDir, err := os.MkdirTemp(outputDir, "download_temp_*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	return &Downloader{
		URL:          url,
		Output:       output,
		Threads:      threads,
		FileSize:     0,
		TempDir:      tempDir,
		TempFiles:    make([]string, 0),
		stopProgress: make(chan struct{}),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		retryCount: 3,
		timeout:    30 * time.Second,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// CheckServerSupport 检查服务器是否支持分段下载并获取文件大小
func (d *Downloader) CheckServerSupport() error {
	req, err := http.NewRequest("HEAD", d.URL, nil)
	if err != nil {
		return fmt.Errorf("创建HEAD请求失败: %v", err)
	}
	req = req.WithContext(d.ctx)

	// 设置User-Agent避免被某些服务器拒绝
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoDownloader/1.0)")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送HEAD请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("服务器返回错误状态码: %d %s", resp.StatusCode, resp.Status)
	}

	// 获取文件大小
	d.FileSize = resp.ContentLength
	if d.FileSize <= 0 {
		fmt.Println("警告：无法获取文件大小，将使用单线程流式下载")
		d.Threads = 1
		return nil
	}

	// 检查是否支持分段下载
	acceptRanges := resp.Header.Get("Accept-Ranges")
	if acceptRanges != "bytes" {
		fmt.Println("警告：服务器不支持分段下载，将使用单线程下载")
		d.Threads = 1
	}

	fmt.Printf("文件大小: %.2f MB\n", float64(d.FileSize)/(1024*1024))
	fmt.Printf("并发线程数: %d\n", d.Threads)
	return nil
}

// DownloadSegment 下载单个文件分段（带重试机制）
func (d *Downloader) DownloadSegment(start, end int64, index int, wg *sync.WaitGroup) error {
	defer wg.Done()

	// 临时文件名（使用固定的文件名，避免重试时创建多个文件）
	tempFile := filepath.Join(d.TempDir, fmt.Sprintf("segment_%d", index))

	// 重试逻辑
	var lastErr error
	for retry := 0; retry <= d.retryCount; retry++ {
		if retry > 0 {
			fmt.Printf("\n分段 %d 重试第 %d 次...\n", index, retry)
			time.Sleep(time.Second * time.Duration(retry)) // 指数退避
			// 删除之前失败的临时文件
			os.Remove(tempFile)
		}

		// 检查上下文是否已取消
		select {
		case <-d.ctx.Done():
			return fmt.Errorf("下载已取消")
		default:
		}

		err := d.downloadSegmentOnce(start, end, index, tempFile)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	// 记录错误
	d.errorsMu.Lock()
	d.errors = append(d.errors, lastErr)
	d.errorsMu.Unlock()

	return lastErr
}

// downloadSegmentOnce 执行单次分段下载
func (d *Downloader) downloadSegmentOnce(start, end int64, index int, tempFile string) error {
	// 创建临时文件
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("创建临时文件 %s 失败: %v", tempFile, err)
	}
	defer func() {
		// 确保数据刷新到磁盘
		f.Sync()
		f.Close()
	}()

	// 构造Range请求头
	req, err := http.NewRequest("GET", d.URL, nil)
	if err != nil {
		return fmt.Errorf("创建GET请求失败: %v", err)
	}
	req = req.WithContext(d.ctx)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoDownloader/1.0)")

	// 只有在多线程模式下才设置Range头
	if d.Threads > 1 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	}

	// 发送请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("下载分段 %d 失败: %v", index, err)
	}
	defer resp.Body.Close()

	// 检查分段响应状态
	if d.Threads > 1 && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("分段 %d 下载失败，服务器返回状态码: %d", index, resp.StatusCode)
	}
	if d.Threads == 1 && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，服务器返回状态码: %d", resp.StatusCode)
	}

	// 读取并写入文件，同时更新进度
	buf := make([]byte, 32*1024) // 增大缓冲区到32KB提升性能
	for {
		// 检查上下文
		select {
		case <-d.ctx.Done():
			return fmt.Errorf("下载已取消")
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			written, writeErr := f.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("写入临时文件 %s 失败: %v", tempFile, writeErr)
			}
			if written != n {
				return fmt.Errorf("写入临时文件 %s 不完整: 期望写入 %d 字节，实际写入 %d 字节", tempFile, n, written)
			}
			// 原子更新已下载字节数
			atomic.AddUint64(&d.progress, uint64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取分段 %d 数据失败: %v", index, err)
		}
	}

	// 验证下载的分段大小（仅在多线程模式下）
	if d.Threads > 1 {
		expectedSize := end - start + 1
		fileInfo, err := f.Stat()
		if err != nil {
			return fmt.Errorf("获取临时文件 %s 信息失败: %v", tempFile, err)
		}
		if fileInfo.Size() != expectedSize {
			return fmt.Errorf("分段 %d 大小不匹配: 期望 %d 字节，实际 %d 字节", index, expectedSize, fileInfo.Size())
		}
	}

	return nil
}

// MergeSegments 合并所有分段文件
func (d *Downloader) MergeSegments() error {
	// 创建最终输出文件
	outFile, err := os.Create(d.Output)
	if err != nil {
		return fmt.Errorf("创建输出文件 %s 失败: %v", d.Output, err)
	}
	defer func() {
		// 确保数据刷新到磁盘
		outFile.Sync()
		outFile.Close()
	}()

	// 按索引顺序合并每个分段（而不是依赖TempFiles的顺序）
	for i := 0; i < d.Threads; i++ {
		tempFile := filepath.Join(d.TempDir, fmt.Sprintf("segment_%d", i))
		
		// 检查临时文件是否存在
		if _, err := os.Stat(tempFile); os.IsNotExist(err) {
			return fmt.Errorf("临时文件 %s 不存在", tempFile)
		}
		
		f, err := os.Open(tempFile)
		if err != nil {
			return fmt.Errorf("打开临时文件 %s 失败: %v", tempFile, err)
		}
		
		// 复制文件内容
		copied, err := io.Copy(outFile, f)
		f.Close()
		
		if err != nil {
			return fmt.Errorf("合并临时文件 %s 失败: %v", tempFile, err)
		}
		
		// 验证复制的字节数
		if d.Threads > 1 {
			fileInfo, _ := os.Stat(tempFile)
			if fileInfo != nil && copied != fileInfo.Size() {
				return fmt.Errorf("合并分段 %d 时数据不完整: 期望 %d 字节，实际 %d 字节", i, fileInfo.Size(), copied)
			}
		}
	}

	return nil
}

// ShowProgress 实时显示下载进度（增强版，显示速度和预计剩余时间）
func (d *Downloader) ShowProgress() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastProgress := uint64(0)
	lastTime := time.Now()

	for {
		select {
		case <-ticker.C:
			d.progressMu.Lock()
			currentProgress := atomic.LoadUint64(&d.progress)
			currentTime := time.Now()
			elapsed := currentTime.Sub(lastTime).Seconds()

			// 计算下载速度
			speed := float64(0)
			if elapsed > 0 {
				speed = float64(currentProgress-lastProgress) / elapsed / 1024 / 1024 // MB/s
			}

			if d.FileSize <= 0 {
				fmt.Printf("\r已下载: %.2f MB | 速度: %.2f MB/s",
					float64(currentProgress)/(1024*1024),
					speed)
			} else {
				percent := float64(currentProgress) / float64(d.FileSize) * 100
				// 计算预计剩余时间
				eta := ""
				if speed > 0 {
					remaining := float64(d.FileSize-int64(currentProgress)) / (speed * 1024 * 1024)
					eta = fmt.Sprintf(" | 预计剩余: %s", formatDuration(time.Duration(remaining)*time.Second))
				}
				fmt.Printf("\r已下载: %.2f MB / %.2f MB (%.2f%%) | 速度: %.2f MB/s%s",
					float64(currentProgress)/(1024*1024),
					float64(d.FileSize)/(1024*1024),
					percent,
					speed,
					eta)
			}

			lastProgress = currentProgress
			lastTime = currentTime
			d.progressMu.Unlock()

		case <-d.stopProgress:
			// 最终进度显示
			d.progressMu.Lock()
			currentProgress := atomic.LoadUint64(&d.progress)
			totalTime := time.Since(d.startTime)
			avgSpeed := float64(currentProgress) / totalTime.Seconds() / 1024 / 1024

			if d.FileSize > 0 {
				percent := float64(currentProgress) / float64(d.FileSize) * 100
				fmt.Printf("\r已下载: %.2f MB / %.2f MB (%.2f%%) | 平均速度: %.2f MB/s | 总耗时: %s\n",
					float64(currentProgress)/(1024*1024),
					float64(d.FileSize)/(1024*1024),
					percent,
					avgSpeed,
					formatDuration(totalTime))
			} else {
				fmt.Printf("\r已下载: %.2f MB | 平均速度: %.2f MB/s | 总耗时: %s\n",
					float64(currentProgress)/(1024*1024),
					avgSpeed,
					formatDuration(totalTime))
			}
			d.progressMu.Unlock()
			return
		}
	}
}

// formatDuration 格式化时间显示
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// Cleanup 清理临时文件和目录
func (d *Downloader) Cleanup() {
	// 取消上下文
	if d.cancel != nil {
		d.cancel()
	}

	// 关闭进度显示
	select {
	case <-d.stopProgress:
		// 已关闭
	default:
		close(d.stopProgress)
	}

	// 删除临时文件（按索引删除，确保所有文件都被清理）
	for i := 0; i < d.Threads; i++ {
		tempFile := filepath.Join(d.TempDir, fmt.Sprintf("segment_%d", i))
		if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
			fmt.Printf("警告：删除临时文件 %s 失败: %v\n", tempFile, err)
		}
	}
	// 删除临时目录
	if err := os.RemoveAll(d.TempDir); err != nil {
		fmt.Printf("警告：删除临时目录 %s 失败: %v\n", d.TempDir, err)
	}
}

// Download 执行下载流程
func (d *Downloader) Download() error {
	defer d.Cleanup()

	// 记录开始时间
	d.startTime = time.Now()

	// 检查服务器支持
	fmt.Println("正在检查服务器支持...")
	if err := d.CheckServerSupport(); err != nil {
		return err
	}

	// 启动进度显示goroutine
	go d.ShowProgress()

	// 单线程下载（不支持分段或文件大小未知）
	if d.Threads == 1 {
		fmt.Println("开始单线程下载...")
		var wg sync.WaitGroup
		wg.Add(1)
		err := d.DownloadSegment(0, d.FileSize-1, 0, &wg)
		wg.Wait()
		if err != nil {
			return fmt.Errorf("下载失败: %v", err)
		}
	} else {
		// 多线程分段下载：计算每个分段的起始和结束位置
		fmt.Printf("开始 %d 线程并发下载...\n", d.Threads)
		segmentSize := d.FileSize / int64(d.Threads)
		var wg sync.WaitGroup
		errChan := make(chan error, d.Threads)

		for i := 0; i < d.Threads; i++ {
			start := int64(i) * segmentSize
			end := start + segmentSize - 1

			// 最后一个分段处理剩余部分
			if i == d.Threads-1 {
				end = d.FileSize - 1
			}

			wg.Add(1)
			go func(idx int, s, e int64) {
				if err := d.DownloadSegment(s, e, idx, &wg); err != nil {
					fmt.Printf("\n分段 %d 下载失败: %v\n", idx, err)
					errChan <- err
					d.cancel() // 取消其他下载
				}
			}(i, start, end)
		}

		// 等待所有分段下载完成
		wg.Wait()
		close(errChan)

		// 检查是否有错误
		if len(errChan) > 0 {
			return <-errChan
		}
	}

	// 检查是否有错误
	d.errorsMu.Lock()
	hasErrors := len(d.errors) > 0
	d.errorsMu.Unlock()

	if hasErrors {
		return fmt.Errorf("下载过程中出现错误，请查看上方日志")
	}

	// 合并分段文件
	fmt.Println("\n开始合并分段文件...")
	if err := d.MergeSegments(); err != nil {
		return fmt.Errorf("合并分段失败: %v", err)
	}

	// 验证文件大小
	if d.FileSize > 0 {
		fileInfo, err := os.Stat(d.Output)
		if err != nil {
			fmt.Printf("警告：无法验证文件大小: %v\n", err)
		} else if fileInfo.Size() != d.FileSize {
			return fmt.Errorf("文件大小不匹配：期望 %d 字节，实际 %d 字节", d.FileSize, fileInfo.Size())
		}
	}

	fmt.Printf("\n✓ 下载完成！文件已保存至: %s\n", d.Output)
	return nil
}

func main() {
	// 解析命令行参数
	url := flag.String("url", "", "下载链接（必填）")
	output := flag.String("output", "download_file", "输出文件名（可选，默认download_file）")
	threads := flag.Int("threads", 5, "并发下载线程数（可选，默认5，范围1-20）")
	retry := flag.Int("retry", 3, "失败重试次数（可选，默认3）")
	timeout := flag.Int("timeout", 30, "请求超时时间（秒，可选，默认30）")
	flag.Parse()

	// 校验必填参数
	if *url == "" {
		fmt.Println("错误：必须指定下载链接（-url 参数）")
		fmt.Println("\n使用示例：")
		fmt.Println("  ./download -url https://example.com/file.zip")
		fmt.Println("  ./download -url https://example.com/file.zip -output myfile.zip -threads 10")
		flag.Usage()
		os.Exit(1)
	}

	// 限制线程数范围
	if *threads < 1 || *threads > 20 {
		fmt.Println("警告：线程数超出合理范围（1-20），自动调整为5")
		*threads = 5
	}

	// 限制重试次数
	if *retry < 0 || *retry > 10 {
		fmt.Println("警告：重试次数超出合理范围（0-10），自动调整为3")
		*retry = 3
	}

	// 限制超时时间
	if *timeout < 5 || *timeout > 300 {
		fmt.Println("警告：超时时间超出合理范围（5-300秒），自动调整为30秒")
		*timeout = 30
	}

	fmt.Println("========================================")
	fmt.Println("      Go 多线程下载工具 v1.0")
	fmt.Println("========================================")
	fmt.Printf("下载链接: %s\n", *url)
	fmt.Printf("输出文件: %s\n", *output)
	fmt.Printf("线程数: %d\n", *threads)
	fmt.Printf("重试次数: %d\n", *retry)
	fmt.Printf("超时时间: %d秒\n", *timeout)
	fmt.Println("========================================\n")

	// 创建下载器并执行下载
	downloader, err := NewDownloader(*url, *output, *threads)
	if err != nil {
		fmt.Printf("✗ 初始化下载器失败: %v\n", err)
		os.Exit(1)
	}

	// 设置重试次数和超时时间
	downloader.retryCount = *retry
	downloader.timeout = time.Duration(*timeout) * time.Second
	downloader.client.Timeout = downloader.timeout

	if err := downloader.Download(); err != nil {
		fmt.Printf("\n✗ 下载失败: %v\n", err)
		os.Exit(1)
	}
}
