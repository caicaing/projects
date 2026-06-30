const MARD_COLORS = Array.isArray(globalThis.MARD_COLORS) ? globalThis.MARD_COLORS : [];

const state = {
  image: null,
  imageUrl: "",
  pattern: null,
  editor: {
    mode: "paint",
    paintCode: "",
    sampleSourceActive: false,
    selectedIndices: new Set(),
    pointerDown: false,
    pointerAction: null,
    selectionAnchor: null,
    lastIndex: null,
    dirty: false,
  },
  sourcePreviewRect: null,
};

const refs = {
  controlsForm: document.getElementById("controlsForm"),
  imageInput: document.getElementById("imageInput"),
  stylePresetSelect: document.getElementById("stylePresetSelect"),
  presetSelect: document.getElementById("presetSelect"),
  widthInput: document.getElementById("widthInput"),
  heightInput: document.getElementById("heightInput"),
  lockAspectInput: document.getElementById("lockAspectInput"),
  maxColorsInput: document.getElementById("maxColorsInput"),
  fitModeInput: document.getElementById("fitModeInput"),
  alphaThresholdInput: document.getElementById("alphaThresholdInput"),
  brightnessInput: document.getElementById("brightnessInput"),
  contrastInput: document.getElementById("contrastInput"),
  saturationInput: document.getElementById("saturationInput"),
  despeckleInput: document.getElementById("despeckleInput"),
  protectDarkLinesInput: document.getElementById("protectDarkLinesInput"),
  ignoreWhiteBgInput: document.getElementById("ignoreWhiteBgInput"),
  mergeSimilarColorsInput: document.getElementById("mergeSimilarColorsInput"),
  mergeToleranceInput: document.getElementById("mergeToleranceInput"),
  mergeTinyRegionsInput: document.getElementById("mergeTinyRegionsInput"),
  tinyRegionSizeInput: document.getElementById("tinyRegionSizeInput"),
  sourceCanvas: document.getElementById("sourceCanvas"),
  patternCanvas: document.getElementById("patternCanvas"),
  editorCanvas: document.getElementById("editorCanvas"),
  statusText: document.getElementById("statusText"),
  sourceMeta: document.getElementById("sourceMeta"),
  patternMeta: document.getElementById("patternMeta"),
  summaryStats: document.getElementById("summaryStats"),
  legendTable: document.getElementById("legendTable"),
  paintColorSelect: document.getElementById("paintColorSelect"),
  editorModeGroup: document.getElementById("editorModeGroup"),
  editorHint: document.getElementById("editorHint"),
  selectionMeta: document.getElementById("selectionMeta"),
  applySelectedBtn: document.getElementById("applySelectedBtn"),
  selectPaintColorBtn: document.getElementById("selectPaintColorBtn"),
  clearSelectionBtn: document.getElementById("clearSelectionBtn"),
  sampleSourceColorBtn: document.getElementById("sampleSourceColorBtn"),
  downloadPngBtn: document.getElementById("downloadPngBtn"),
  downloadCsvBtn: document.getElementById("downloadCsvBtn"),
  printBtn: document.getElementById("printBtn"),
};

const palette = MARD_COLORS.map((entry) => {
  const rgb = hexToRgb(entry.hex);
  return {
    ...entry,
    rgb,
    lab: rgbToLab(rgb.r, rgb.g, rgb.b),
  };
});

const paletteByCode = new Map(palette.map((entry) => [entry.code, entry]));
const stylePresets = {
  original: {
    label: "原图优先",
    maxColors: 24,
    brightness: 100,
    contrast: 100,
    saturation: 100,
    despeckle: true,
    protectDarkLines: true,
    ignoreWhiteBg: true,
    mergeSimilarColors: true,
    mergeTolerance: 8,
    mergeTinyRegions: true,
    tinyRegionSize: 4,
  },
  cartoon: {
    label: "卡通优先",
    maxColors: 16,
    brightness: 100,
    contrast: 104,
    saturation: 104,
    despeckle: true,
    protectDarkLines: true,
    ignoreWhiteBg: true,
    mergeSimilarColors: true,
    mergeTolerance: 9,
    mergeTinyRegions: true,
    tinyRegionSize: 5,
  },
  lineart: {
    label: "线稿优先",
    maxColors: 12,
    brightness: 100,
    contrast: 108,
    saturation: 96,
    despeckle: true,
    protectDarkLines: true,
    ignoreWhiteBg: true,
    mergeSimilarColors: true,
    mergeTolerance: 10,
    mergeTinyRegions: true,
    tinyRegionSize: 6,
  },
};

bindEvents();
renderEmptyCanvases();
renderEmptyEditor();
applyStylePreset(refs.stylePresetSelect.value, { silent: true });
setStatus(`已加载 MARD ${palette.length} 色。上传图片后开始生成。`);

function bindEvents() {
  refs.imageInput.addEventListener("change", handleImageChange);
  refs.stylePresetSelect.addEventListener("change", handleStylePresetChange);
  refs.presetSelect.addEventListener("change", handlePresetChange);
  refs.lockAspectInput.addEventListener("change", syncHeightWithAspect);
  refs.widthInput.addEventListener("input", syncHeightWithAspect);
  refs.controlsForm.addEventListener("submit", handleGenerate);
  refs.downloadPngBtn.addEventListener("click", downloadPng);
  refs.downloadCsvBtn.addEventListener("click", downloadCsv);
  refs.printBtn.addEventListener("click", () => window.print());
  refs.sampleSourceColorBtn.addEventListener("click", toggleSourceColorSampling);

  refs.paintColorSelect.addEventListener("change", () => {
    state.editor.paintCode = refs.paintColorSelect.value;
    renderEditorControls();
  });

  refs.editorModeGroup.addEventListener("click", (event) => {
    const button = event.target.closest("[data-mode]");
    if (!button) {
      return;
    }
    state.editor.mode = button.dataset.mode === "select" ? "select" : "paint";
    renderEditorControls();
  });

  refs.applySelectedBtn.addEventListener("click", applySelectedCells);
  refs.selectPaintColorBtn.addEventListener("click", selectCellsByPaintColor);
  refs.clearSelectionBtn.addEventListener("click", clearSelection);

  refs.legendTable.addEventListener("click", handleLegendTableClick);

  refs.sourceCanvas.addEventListener("click", handleSourceCanvasClick);
  refs.editorCanvas.addEventListener("pointerdown", handleEditorPointerDown);
  refs.editorCanvas.addEventListener("pointermove", handleEditorPointerMove);
  refs.editorCanvas.addEventListener("pointerleave", handleEditorPointerLeave);
  if (typeof window.addEventListener === "function") {
    window.addEventListener("pointerup", handleEditorPointerUp);
  }

  [
    refs.maxColorsInput,
    refs.brightnessInput,
    refs.contrastInput,
    refs.saturationInput,
    refs.despeckleInput,
    refs.protectDarkLinesInput,
    refs.ignoreWhiteBgInput,
    refs.mergeSimilarColorsInput,
    refs.mergeToleranceInput,
    refs.mergeTinyRegionsInput,
    refs.tinyRegionSizeInput,
  ].forEach((element) => {
    const eventName = element.type === "range" || element.type === "number" ? "input" : "change";
    element.addEventListener(eventName, markStylePresetAsCustom);
  });
}

function handlePresetChange() {
  if (refs.presetSelect.value === "custom") {
    return;
  }
  refs.widthInput.value = refs.presetSelect.value;
  syncHeightWithAspect();
}

function handleStylePresetChange() {
  applyStylePreset(refs.stylePresetSelect.value);
}

function applyStylePreset(presetKey, options = {}) {
  if (presetKey === "custom") {
    if (!options.silent) {
      setStatus("已切换到自定义参数。");
    }
    return;
  }

  const preset = stylePresets[presetKey] || stylePresets.original;
  refs.maxColorsInput.value = String(preset.maxColors);
  refs.brightnessInput.value = String(preset.brightness);
  refs.contrastInput.value = String(preset.contrast);
  refs.saturationInput.value = String(preset.saturation);
  refs.despeckleInput.checked = preset.despeckle;
  refs.protectDarkLinesInput.checked = preset.protectDarkLines;
  refs.ignoreWhiteBgInput.checked = preset.ignoreWhiteBg;
  refs.mergeSimilarColorsInput.checked = preset.mergeSimilarColors;
  refs.mergeToleranceInput.value = String(preset.mergeTolerance);
  refs.mergeTinyRegionsInput.checked = preset.mergeTinyRegions;
  refs.tinyRegionSizeInput.value = String(preset.tinyRegionSize);

  if (!options.silent) {
    setStatus(`已应用“${preset.label}”预设。`);
  }
}

function markStylePresetAsCustom() {
  const currentKey = refs.stylePresetSelect.value;
  if (currentKey === "custom") {
    return;
  }

  const preset = stylePresets[currentKey];
  if (!preset) {
    refs.stylePresetSelect.value = "custom";
    return;
  }

  const matchesPreset = (
    Number.parseInt(refs.maxColorsInput.value, 10) === preset.maxColors
    && Number.parseInt(refs.brightnessInput.value, 10) === preset.brightness
    && Number.parseInt(refs.contrastInput.value, 10) === preset.contrast
    && Number.parseInt(refs.saturationInput.value, 10) === preset.saturation
    && refs.despeckleInput.checked === preset.despeckle
    && refs.protectDarkLinesInput.checked === preset.protectDarkLines
    && refs.ignoreWhiteBgInput.checked === preset.ignoreWhiteBg
    && refs.mergeSimilarColorsInput.checked === preset.mergeSimilarColors
    && Number.parseInt(refs.mergeToleranceInput.value, 10) === preset.mergeTolerance
    && refs.mergeTinyRegionsInput.checked === preset.mergeTinyRegions
    && Number.parseInt(refs.tinyRegionSizeInput.value, 10) === preset.tinyRegionSize
  );

  if (!matchesPreset) {
    refs.stylePresetSelect.value = "custom";
  }
}

function handleImageChange(event) {
  const [file] = event.target.files || [];
  if (!file) {
    return;
  }

  if (state.imageUrl) {
    URL.revokeObjectURL(state.imageUrl);
  }

  const imageUrl = URL.createObjectURL(file);
  const image = new Image();
  image.onload = () => {
    state.editor.sampleSourceActive = false;
    state.image = image;
    state.imageUrl = imageUrl;
    refs.sourceMeta.textContent = `${image.naturalWidth} × ${image.naturalHeight}`;
    drawImagePreview(image);
    syncHeightWithAspect();
    setStatus("图片已加载，调整参数后点击生成图纸。");
  };
  image.onerror = () => {
    setStatus("图片加载失败，请换一张图片再试。");
  };
  image.src = imageUrl;
}

function syncHeightWithAspect() {
  if (!state.image || !refs.lockAspectInput.checked) {
    return;
  }

  const width = clampInt(refs.widthInput.value, 8, 120, 32);
  const ratio = state.image.naturalHeight / state.image.naturalWidth;
  const height = Math.max(1, Math.round(width * ratio));
  refs.heightInput.value = String(clampInt(height, 8, 120, 32));
}

function handleGenerate(event) {
  event.preventDefault();

  if (!state.image) {
    setStatus("请先上传图片。");
    return;
  }

  const settings = readSettings();
  setStatus("正在生成拼豆图纸，请稍等...");

  window.requestAnimationFrame(() => {
    try {
      const pattern = buildPattern(state.image, settings);
      state.pattern = pattern;
      initializeEditorState(pattern);
      refreshPatternViews("生成完成。现在可以继续手工改色和批量编辑。");
      refs.downloadPngBtn.disabled = false;
      refs.downloadCsvBtn.disabled = false;
      refs.printBtn.disabled = false;
    } catch (error) {
      console.error(error);
      setStatus(`生成失败：${error.message}`);
    }
  });
}

function readSettings() {
  const width = clampInt(refs.widthInput.value, 8, 120, 32);
  const height = clampInt(refs.heightInput.value, 8, 120, 32);
  return {
    width,
    height,
    maxColors: clampInt(refs.maxColorsInput.value, 4, 64, 18),
    fitMode: refs.fitModeInput.value === "cover" ? "cover" : "contain",
    alphaThreshold: clampInt(refs.alphaThresholdInput.value, 0, 255, 24),
    brightness: clampInt(refs.brightnessInput.value, 60, 140, 100) / 100,
    contrast: clampInt(refs.contrastInput.value, 60, 160, 100) / 100,
    saturation: clampInt(refs.saturationInput.value, 40, 160, 100) / 100,
    despeckle: refs.despeckleInput.checked,
    protectDarkLines: refs.protectDarkLinesInput.checked,
    ignoreWhiteBg: refs.ignoreWhiteBgInput.checked,
    mergeSimilarColors: refs.mergeSimilarColorsInput.checked,
    mergeTolerance: clampInt(refs.mergeToleranceInput.value, 2, 20, 8),
    mergeTinyRegions: refs.mergeTinyRegionsInput.checked,
    tinyRegionSize: clampInt(refs.tinyRegionSizeInput.value, 1, 12, 4),
  };
}

function buildPattern(image, settings) {
  const preparedCanvas = document.createElement("canvas");
  preparedCanvas.width = settings.width;
  preparedCanvas.height = settings.height;
  const ctx = preparedCanvas.getContext("2d", { willReadFrequently: true });
  ctx.clearRect(0, 0, preparedCanvas.width, preparedCanvas.height);
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.filter = `brightness(${settings.brightness}) contrast(${settings.contrast}) saturate(${settings.saturation})`;

  const drawRect = getDrawRect(image.naturalWidth, image.naturalHeight, preparedCanvas.width, preparedCanvas.height, settings.fitMode);
  ctx.drawImage(image, drawRect.x, drawRect.y, drawRect.width, drawRect.height);
  ctx.filter = "none";

  const imageData = ctx.getImageData(0, 0, preparedCanvas.width, preparedCanvas.height);
  const pixels = extractPixels(imageData, settings.alphaThreshold);
  if (settings.ignoreWhiteBg) {
    removeWhiteBackgroundFromEdges(pixels, preparedCanvas.width, preparedCanvas.height);
  }
  const visiblePixels = pixels.filter((pixel) => pixel.a > 0);

  if (!visiblePixels.length) {
    throw new Error("透明区域过多，未检测到可用像素。");
  }

  const quantized = quantizeVisiblePixels(visiblePixels, settings.maxColors);
  const clusterPalette = mapClustersToPalette(quantized.clusters);
  const grid = pixels.map((pixel) => {
    if (!pixel.a) {
      return null;
    }
    if (settings.protectDarkLines && isProtectedDarkPixel(pixel.lab)) {
      return findProtectedDarkMatch(pixel.lab);
    }
    const clusterIndex = quantized.assignmentsByIndex.get(pixel.index);
    return clusterPalette[clusterIndex] || null;
  });

  const cleanedGrid = settings.despeckle
    ? despeckleGrid(grid, preparedCanvas.width, preparedCanvas.height)
    : grid;
  const normalizedGrid = normalizePatternGrid(
    cleanedGrid,
    preparedCanvas.width,
    preparedCanvas.height,
    settings
  );

  const pattern = {
    width: preparedCanvas.width,
    height: preparedCanvas.height,
    grid: [...normalizedGrid],
    settings,
    legend: [],
    totalBeads: 0,
    boardCount: "",
    rendered: null,
    editorMetrics: null,
  };

  recalculatePattern(pattern);
  return pattern;
}

function initializeEditorState(pattern) {
  state.editor.sampleSourceActive = false;
  state.editor.selectedIndices = new Set();
  state.editor.pointerDown = false;
  state.editor.pointerAction = null;
  state.editor.selectionAnchor = null;
  state.editor.lastIndex = null;
  state.editor.dirty = false;
  state.editor.paintCode = pattern.legend[0]?.code || palette[0]?.code || "";
}

function recalculatePattern(pattern) {
  pattern.legend = buildLegend(pattern.grid);
  pattern.totalBeads = pattern.legend.reduce((sum, item) => sum + item.count, 0);
  pattern.boardCount = `${Math.ceil(pattern.width / 29)} × ${Math.ceil(pattern.height / 29)}`;
  pattern.rendered = renderPatternCanvas(pattern.grid, pattern.width, pattern.height, pattern.legend);
}

function refreshPatternViews(statusText) {
  if (!state.pattern) {
    return;
  }

  recalculatePattern(state.pattern);
  normalizePaintColor();
  renderPattern(state.pattern);
  renderSummary(state.pattern);
  renderEditor(state.pattern);
  renderEditorControls();
  refs.patternMeta.textContent = `${state.pattern.width} × ${state.pattern.height} / ${state.pattern.legend.length} 色`;

  if (statusText) {
    setStatus(`${statusText} 当前共 ${state.pattern.totalBeads} 颗拼豆，${state.pattern.legend.length} 种颜色。`);
  }
}

function normalizePaintColor() {
  if (state.editor.paintCode && paletteByCode.has(state.editor.paintCode)) {
    return;
  }
  state.editor.paintCode = state.pattern?.legend[0]?.code || palette[0]?.code || "";
}

function extractPixels(imageData, alphaThreshold) {
  const pixels = [];
  const { data, width } = imageData;
  for (let i = 0; i < data.length; i += 4) {
    const alpha = data[i + 3];
    const index = i / 4;
    if (alpha <= alphaThreshold) {
      pixels.push({ index, a: 0 });
      continue;
    }

    pixels.push({
      index,
      a: alpha,
      x: index % width,
      y: Math.floor(index / width),
      r: data[i],
      g: data[i + 1],
      b: data[i + 2],
      lab: rgbToLab(data[i], data[i + 1], data[i + 2]),
    });
  }
  return pixels;
}

function removeWhiteBackgroundFromEdges(pixels, width, height) {
  const queue = [];
  const visited = new Uint8Array(pixels.length);
  const edgeIndices = new Set();

  for (let x = 0; x < width; x += 1) {
    edgeIndices.add(x);
    edgeIndices.add(((height - 1) * width) + x);
  }
  for (let y = 0; y < height; y += 1) {
    edgeIndices.add(y * width);
    edgeIndices.add((y * width) + (width - 1));
  }

  edgeIndices.forEach((index) => {
    const pixel = pixels[index];
    if (pixel?.a > 0 && isNearWhiteBackgroundPixel(pixel.lab)) {
      queue.push(index);
      visited[index] = 1;
    }
  });

  const neighbors = [
    [-1, 0], [1, 0], [0, -1], [0, 1],
  ];

  while (queue.length) {
    const index = queue.shift();
    const pixel = pixels[index];
    if (!pixel || pixel.a <= 0) {
      continue;
    }

    pixel.a = 0;
    const x = index % width;
    const y = Math.floor(index / width);

    for (const [dx, dy] of neighbors) {
      const nx = x + dx;
      const ny = y + dy;
      if (nx < 0 || ny < 0 || nx >= width || ny >= height) {
        continue;
      }
      const nextIndex = ny * width + nx;
      if (visited[nextIndex]) {
        continue;
      }
      const nextPixel = pixels[nextIndex];
      if (nextPixel?.a > 0 && isNearWhiteBackgroundPixel(nextPixel.lab)) {
        visited[nextIndex] = 1;
        queue.push(nextIndex);
      }
    }
  }
}

function isNearWhiteBackgroundPixel(lab) {
  const chroma = Math.sqrt((lab.a ** 2) + (lab.b ** 2));
  return lab.l >= 93 && chroma <= 10;
}

function quantizeVisiblePixels(visiblePixels, maxColors) {
  const uniqueMap = new Map();
  for (const pixel of visiblePixels) {
    const key = `${pixel.r},${pixel.g},${pixel.b}`;
    const existing = uniqueMap.get(key);
    if (existing) {
      existing.count += 1;
      existing.indices.push(pixel.index);
    } else {
      uniqueMap.set(key, {
        key,
        count: 1,
        indices: [pixel.index],
        lab: pixel.lab,
      });
    }
  }

  const samples = [...uniqueMap.values()];
  if (samples.length <= maxColors) {
    const assignmentsByIndex = new Map();
    samples.forEach((sample, sampleIndex) => {
      sample.indices.forEach((pixelIndex) => assignmentsByIndex.set(pixelIndex, sampleIndex));
    });
    return {
      clusters: samples.map((sample) => sample.lab),
      assignmentsByIndex,
    };
  }

  let centroids = initCentroids(samples, maxColors);
  const assignments = new Array(samples.length).fill(0);

  for (let iteration = 0; iteration < 10; iteration += 1) {
    let changed = false;
    const sums = centroids.map(() => ({ l: 0, a: 0, b: 0, count: 0 }));

    samples.forEach((sample, sampleIndex) => {
      const nextClusterIndex = nearestLabIndex(sample.lab, centroids);
      if (assignments[sampleIndex] !== nextClusterIndex) {
        changed = true;
        assignments[sampleIndex] = nextClusterIndex;
      }
      const bucket = sums[nextClusterIndex];
      bucket.l += sample.lab.l * sample.count;
      bucket.a += sample.lab.a * sample.count;
      bucket.b += sample.lab.b * sample.count;
      bucket.count += sample.count;
    });

    centroids = centroids.map((centroid, index) => {
      const bucket = sums[index];
      if (!bucket.count) {
        return samples[index % samples.length].lab;
      }
      return {
        l: bucket.l / bucket.count,
        a: bucket.a / bucket.count,
        b: bucket.b / bucket.count,
      };
    });

    if (!changed && iteration > 0) {
      break;
    }
  }

  const assignmentsByIndex = new Map();
  samples.forEach((sample, sampleIndex) => {
    const clusterIndex = assignments[sampleIndex];
    sample.indices.forEach((pixelIndex) => assignmentsByIndex.set(pixelIndex, clusterIndex));
  });

  return { clusters: centroids, assignmentsByIndex };
}

function initCentroids(samples, maxColors) {
  const sorted = [...samples].sort((left, right) => right.count - left.count);
  const centroids = [sorted[0].lab];

  while (centroids.length < maxColors) {
    let bestSample = sorted[centroids.length % sorted.length];
    let bestDistance = -1;

    for (const sample of sorted) {
      const distance = nearestLabDistance(sample.lab, centroids);
      if (distance > bestDistance) {
        bestDistance = distance;
        bestSample = sample;
      }
    }

    centroids.push(bestSample.lab);
  }

  return centroids;
}

function mapClustersToPalette(clusters) {
  return clusters.map((cluster) => findBestPaletteMatch(cluster));
}

function findBestPaletteMatch(cluster) {
  let best = palette[0];
  let bestScore = Infinity;

  for (const entry of palette) {
    const score = paletteMatchScore(cluster, entry);
    if (score < bestScore) {
      best = entry;
      bestScore = score;
    }
  }

  return best;
}

function isProtectedDarkPixel(lab) {
  const chroma = Math.sqrt((lab.a ** 2) + (lab.b ** 2));
  return lab.l <= 30 && chroma <= 16;
}

function findProtectedDarkMatch(lab) {
  const preferredCodes = ["H7", "H16", "H6", "C18", "H5"];
  let best = paletteByCode.get(preferredCodes[0]) || palette[0];
  let bestScore = Infinity;

  for (const code of preferredCodes) {
    const entry = paletteByCode.get(code);
    if (!entry) {
      continue;
    }
    const score = deltaE2000(lab, entry.lab);
    if (score < bestScore) {
      best = entry;
      bestScore = score;
    }
  }

  return best;
}

function paletteMatchScore(sourceLab, paletteEntry) {
  const base = deltaE2000(sourceLab, paletteEntry.lab);
  const sourceChroma = Math.sqrt((sourceLab.a ** 2) + (sourceLab.b ** 2));
  const targetChroma = Math.sqrt((paletteEntry.lab.a ** 2) + (paletteEntry.lab.b ** 2));
  let penalty = 0;

  // Near-neutral colors should prefer black/white/gray/brownish neutrals,
  // instead of drifting into green or blue simply because of centroid noise.
  if (sourceChroma < 10 && targetChroma > 18) {
    penalty += 12 + ((targetChroma - 18) * 0.18);
  }

  // Very dark areas are usually outlines, pupils, or shadows.
  // Keep them anchored to dark swatches instead of brighter dark colors.
  if (sourceLab.l < 22 && paletteEntry.lab.l > 34) {
    penalty += 10 + ((paletteEntry.lab.l - 34) * 0.35);
  }

  // Very bright pixels should stay close to white/light tones.
  if (sourceLab.l > 92 && paletteEntry.lab.l < 82) {
    penalty += 8 + ((82 - paletteEntry.lab.l) * 0.25);
  }

  return base + penalty;
}

function despeckleGrid(grid, width, height) {
  const next = [...grid];
  const offsets = [
    [-1, -1], [0, -1], [1, -1],
    [-1, 0],           [1, 0],
    [-1, 1],  [0, 1],  [1, 1],
  ];

  for (let y = 0; y < height; y += 1) {
    for (let x = 0; x < width; x += 1) {
      const index = y * width + x;
      const current = grid[index];
      if (!current) {
        continue;
      }

      const counts = new Map();
      let visibleNeighbors = 0;
      for (const [dx, dy] of offsets) {
        const nx = x + dx;
        const ny = y + dy;
        if (nx < 0 || ny < 0 || nx >= width || ny >= height) {
          continue;
        }
        const neighbor = grid[ny * width + nx];
        if (!neighbor) {
          continue;
        }
        visibleNeighbors += 1;
        counts.set(neighbor.code, (counts.get(neighbor.code) || 0) + 1);
      }

      if (visibleNeighbors < 5) {
        continue;
      }

      let dominantCode = current.code;
      let dominantCount = 0;
      for (const [code, count] of counts.entries()) {
        if (count > dominantCount) {
          dominantCode = code;
          dominantCount = count;
        }
      }

      const currentNeighborCount = counts.get(current.code) || 0;
      if (dominantCode !== current.code && dominantCount >= 5 && currentNeighborCount <= 1) {
        next[index] = paletteByCode.get(dominantCode) || current;
      }
    }
  }
  return next;
}

function buildLegend(grid) {
  const counts = new Map();
  for (const cell of grid) {
    if (!cell) {
      continue;
    }
    const existing = counts.get(cell.code);
    if (existing) {
      existing.count += 1;
    } else {
      counts.set(cell.code, {
        code: cell.code,
        hex: cell.hex,
        count: 1,
      });
    }
  }

  return [...counts.values()].sort((left, right) => {
    if (right.count !== left.count) {
      return right.count - left.count;
    }
    return left.code.localeCompare(right.code, "zh-CN");
  });
}

function normalizePatternGrid(grid, width, height, settings) {
  let next = [...grid];

  if (settings.mergeSimilarColors) {
    next = mergeSimilarPaletteCodes(next, settings.mergeTolerance);
  }

  if (settings.mergeTinyRegions) {
    next = mergeTinyColorRegions(next, width, height, settings.tinyRegionSize, settings.mergeTolerance);
  }

  if (settings.mergeSimilarColors) {
    next = mergeSimilarPaletteCodes(next, settings.mergeTolerance);
  }

  return next;
}

function mergeSimilarPaletteCodes(grid, tolerance) {
  const legend = buildLegend(grid);
  if (legend.length < 2) {
    return grid;
  }

  const replacementMap = new Map();
  for (let index = legend.length - 1; index >= 0; index -= 1) {
    const sourceItem = legend[index];
    const sourceEntry = paletteByCode.get(sourceItem.code);
    if (!sourceEntry) {
      continue;
    }

    let bestTarget = null;
    let bestScore = Infinity;
    for (let targetIndex = 0; targetIndex < index; targetIndex += 1) {
      const targetItem = legend[targetIndex];
      const targetEntry = paletteByCode.get(targetItem.code);
      if (!targetEntry || !canMergePaletteEntries(sourceEntry, targetEntry, tolerance)) {
        continue;
      }

      const score = deltaE2000(sourceEntry.lab, targetEntry.lab);
      if (score < bestScore) {
        bestScore = score;
        bestTarget = targetEntry;
      }
    }

    if (bestTarget) {
      replacementMap.set(sourceEntry.code, bestTarget);
    }
  }

  if (!replacementMap.size) {
    return grid;
  }

  return grid.map((cell) => {
    if (!cell) {
      return null;
    }
    return replacementMap.get(cell.code) || cell;
  });
}

function mergeTinyColorRegions(grid, width, height, maxSize, tolerance) {
  const next = [...grid];
  const visited = new Uint8Array(grid.length);
  const neighbors = [
    [-1, 0], [1, 0], [0, -1], [0, 1],
  ];

  for (let index = 0; index < grid.length; index += 1) {
    if (visited[index] || !grid[index]) {
      continue;
    }

    const sourceCode = grid[index].code;
    const queue = [index];
    const region = [];
    visited[index] = 1;

    while (queue.length) {
      const current = queue.pop();
      region.push(current);
      const x = current % width;
      const y = Math.floor(current / width);

      for (const [dx, dy] of neighbors) {
        const nx = x + dx;
        const ny = y + dy;
        if (nx < 0 || ny < 0 || nx >= width || ny >= height) {
          continue;
        }
        const nextIndex = ny * width + nx;
        if (visited[nextIndex] || !grid[nextIndex] || grid[nextIndex].code !== sourceCode) {
          continue;
        }
        visited[nextIndex] = 1;
        queue.push(nextIndex);
      }
    }

    if (region.length > maxSize) {
      continue;
    }

    const sourceEntry = paletteByCode.get(sourceCode);
    const boundaryScores = new Map();
    region.forEach((cellIndex) => {
      const x = cellIndex % width;
      const y = Math.floor(cellIndex / width);
      for (const [dx, dy] of neighbors) {
        const nx = x + dx;
        const ny = y + dy;
        if (nx < 0 || ny < 0 || nx >= width || ny >= height) {
          continue;
        }
        const neighbor = grid[(ny * width) + nx];
        if (!neighbor || neighbor.code === sourceCode) {
          continue;
        }
        const targetEntry = paletteByCode.get(neighbor.code);
        if (!targetEntry) {
          continue;
        }
        const current = boundaryScores.get(neighbor.code) || { weight: 0, entry: targetEntry };
        current.weight += 1;
        boundaryScores.set(neighbor.code, current);
      }
    });

    let bestTarget = null;
    let bestBoundaryScore = -1;
    let bestDistance = Infinity;
    for (const { weight, entry } of boundaryScores.values()) {
      if (!sourceEntry || !canMergePaletteEntries(sourceEntry, entry, tolerance + 2)) {
        continue;
      }
      const distance = deltaE2000(sourceEntry.lab, entry.lab);
      if (weight > bestBoundaryScore || (weight === bestBoundaryScore && distance < bestDistance)) {
        bestTarget = entry;
        bestBoundaryScore = weight;
        bestDistance = distance;
      }
    }

    if (!bestTarget && boundaryScores.size) {
      bestTarget = [...boundaryScores.values()].sort((left, right) => right.weight - left.weight)[0].entry;
    }

    if (!bestTarget) {
      continue;
    }

    region.forEach((cellIndex) => {
      next[cellIndex] = bestTarget;
    });
  }

  return next;
}

function canMergePaletteEntries(sourceEntry, targetEntry, tolerance) {
  const distance = deltaE2000(sourceEntry.lab, targetEntry.lab);
  if (distance > tolerance) {
    return false;
  }

  const lightnessDiff = Math.abs(sourceEntry.lab.l - targetEntry.lab.l);
  if (lightnessDiff > 18) {
    return false;
  }

  const sourceChroma = Math.sqrt((sourceEntry.lab.a ** 2) + (sourceEntry.lab.b ** 2));
  const targetChroma = Math.sqrt((targetEntry.lab.a ** 2) + (targetEntry.lab.b ** 2));
  const oneNeutralOneColorful = Math.min(sourceChroma, targetChroma) < 8 && Math.max(sourceChroma, targetChroma) > 18;
  if (oneNeutralOneColorful) {
    return false;
  }

  return true;
}

function renderPattern(pattern) {
  const ctx = refs.patternCanvas.getContext("2d");
  refs.patternCanvas.width = pattern.rendered.width;
  refs.patternCanvas.height = pattern.rendered.height;
  ctx.clearRect(0, 0, pattern.rendered.width, pattern.rendered.height);
  ctx.drawImage(pattern.rendered, 0, 0);
}

function renderSummary(pattern) {
  const density = pattern.legend.length
    ? `${Math.round((pattern.totalBeads / (pattern.width * pattern.height)) * 100)}%`
    : "0%";

  refs.summaryStats.innerHTML = [
    statItem("图纸尺寸", `${pattern.width} × ${pattern.height}`),
    statItem("总颗数", `${pattern.totalBeads} 颗`),
    statItem("使用色数", `${pattern.legend.length} 色`),
    statItem("推荐板数", `${pattern.boardCount} 块 29×29`),
    statItem("透明留白", `${pattern.width * pattern.height - pattern.totalBeads} 格`),
    statItem("填充密度", density),
  ].join("");

  if (!pattern.legend.length) {
    refs.legendTable.className = "legend-table empty-state";
    refs.legendTable.textContent = "没有可显示的色号。";
    return;
  }

  refs.legendTable.className = "legend-table";
  refs.legendTable.innerHTML = `
    <table>
      <thead>
        <tr>
          <th>色号</th>
          <th>HEX</th>
          <th>颗数</th>
          <th>占比</th>
          <th>编辑</th>
        </tr>
      </thead>
      <tbody>
        ${pattern.legend.map((item) => `
          <tr>
            <td>
              <span class="legend-swatch" style="background:${item.hex}"></span>${item.code}
            </td>
            <td>${item.hex}</td>
            <td>${item.count}</td>
            <td>${((item.count / pattern.totalBeads) * 100).toFixed(1)}%</td>
            <td class="legend-edit-cell">
              <div class="legend-edit-controls">
                <button type="button" class="mini-btn alt" data-action="set-paint" data-code="${item.code}">设为画笔</button>
                <button type="button" class="mini-btn alt" data-action="select-color" data-code="${item.code}">选中此色</button>
                <select class="legend-replace-select" data-source-code="${item.code}">
                  ${buildPaletteOptionsMarkup(item.code)}
                </select>
                <button type="button" class="mini-btn" data-action="replace-all" data-code="${item.code}">整色替换</button>
              </div>
            </td>
          </tr>
        `).join("")}
      </tbody>
    </table>
  `;
}

function renderEditor(pattern) {
  const canvas = refs.editorCanvas;
  const ctx = canvas.getContext("2d");
  const cell = Math.max(18, Math.min(34, Math.floor(1100 / Math.max(pattern.width, pattern.height))));
  const topPad = 52;
  const leftPad = 56;
  const rightPad = 20;
  const bottomPad = 20;
  const gridWidth = pattern.width * cell;
  const gridHeight = pattern.height * cell;

  canvas.width = leftPad + gridWidth + rightPad;
  canvas.height = topPad + gridHeight + bottomPad;
  pattern.editorMetrics = { cell, topPad, leftPad, rightPad, bottomPad };

  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = "#fffdf8";
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  ctx.fillStyle = "#6c523a";
  ctx.font = "12px 'Avenir Next', 'PingFang SC', sans-serif";
  ctx.textAlign = "center";
  for (let x = 0; x < pattern.width; x += 1) {
    ctx.fillText(String(x + 1), leftPad + x * cell + cell / 2, topPad - 14);
  }

  ctx.textAlign = "right";
  for (let y = 0; y < pattern.height; y += 1) {
    ctx.fillText(String(y + 1), leftPad - 10, topPad + y * cell + cell / 2 + 4);
  }

  ctx.textAlign = "center";
  for (let y = 0; y < pattern.height; y += 1) {
    for (let x = 0; x < pattern.width; x += 1) {
      const index = y * pattern.width + x;
      const cellData = pattern.grid[index];
      const left = leftPad + x * cell;
      const top = topPad + y * cell;

      ctx.fillStyle = cellData ? cellData.hex : "#f2ece2";
      ctx.fillRect(left, top, cell, cell);

      ctx.strokeStyle = state.editor.selectedIndices.has(index) ? "#9c2f18" : "#ccb7a0";
      ctx.lineWidth = state.editor.selectedIndices.has(index) ? 2 : 1;
      ctx.strokeRect(left, top, cell, cell);

      if (!cellData) {
        continue;
      }

      const textColor = getReadableTextColor(cellData.rgb);
      ctx.fillStyle = textColor;
      ctx.font = `${Math.max(8, Math.floor(cell * 0.34))}px 'Avenir Next', 'PingFang SC', sans-serif`;
      ctx.fillText(cellData.code, left + cell / 2, top + cell / 2 + Math.max(3, cell * 0.1));
    }
  }
}

function renderEditorControls() {
  const hasPattern = Boolean(state.pattern);
  refs.paintColorSelect.disabled = !hasPattern;
  refs.applySelectedBtn.disabled = !hasPattern || state.editor.selectedIndices.size === 0;
  refs.selectPaintColorBtn.disabled = !hasPattern || !state.editor.paintCode;
  refs.clearSelectionBtn.disabled = !hasPattern || state.editor.selectedIndices.size === 0;
  refs.sampleSourceColorBtn.disabled = !hasPattern || !state.image;
  if (refs.sampleSourceColorBtn.classList) {
    refs.sampleSourceColorBtn.classList.toggle("is-active", state.editor.sampleSourceActive);
  }

  refs.paintColorSelect.innerHTML = hasPattern ? buildPaletteOptionsMarkup(state.editor.paintCode) : "";
  refs.paintColorSelect.value = state.editor.paintCode;

  const modeButtons = typeof refs.editorModeGroup.querySelectorAll === "function"
    ? refs.editorModeGroup.querySelectorAll("[data-mode]")
    : [];
  modeButtons.forEach((button) => {
    button.classList.toggle("is-active", button.dataset.mode === state.editor.mode);
  });

  if (!hasPattern) {
    refs.selectionMeta.textContent = "未选中格子";
    refs.editorHint.textContent = "生成图纸后，可单格修改，也可框选后批量改色。";
    return;
  }

  refs.selectionMeta.textContent = state.editor.selectedIndices.size
    ? `已选 ${state.editor.selectedIndices.size} 格`
    : "未选中格子";

  refs.editorHint.textContent = state.editor.mode === "paint"
    ? "刷格模式：点击或拖拽即可把格子改成当前画笔色号。"
    : "框选模式：拖拽框选一片格子，再点“填充选中格”批量改色。";

  if (state.editor.sampleSourceActive) {
    refs.editorHint.textContent = "原图吸色已开启：请点击左侧原图，自动选取最接近的 MARD 色号。";
  }
}

function renderPatternCanvas(grid, width, height, legend) {
  const cell = Math.max(18, Math.min(36, Math.floor(1200 / Math.max(width, height))));
  const topPad = 64;
  const leftPad = 64;
  const rightPad = 36;
  const bottomPad = 40;
  const gridWidth = width * cell;
  const gridHeight = height * cell;
  const legendColumns = legend.length > 18 ? 2 : 1;
  const legendColumnWidth = 310;
  const legendRowHeight = 38;
  const legendRows = Math.max(1, Math.ceil(legend.length / legendColumns));
  const legendHeight = 108 + legendRows * legendRowHeight;
  const canvas = document.createElement("canvas");
  canvas.width = leftPad + gridWidth + rightPad;
  canvas.height = topPad + gridHeight + bottomPad + legendHeight;
  const ctx = canvas.getContext("2d");

  ctx.fillStyle = "#fffaf5";
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  ctx.fillStyle = "#2f241a";
  ctx.font = "700 30px 'Avenir Next', 'PingFang SC', sans-serif";
  ctx.fillText("拼豆图纸", leftPad, 38);

  ctx.fillStyle = "#6c523a";
  ctx.font = "16px 'Avenir Next', 'PingFang SC', sans-serif";
  ctx.fillText(`${width} × ${height} 格 / MARD ${legend.length} 色`, leftPad, 58);

  for (let x = 0; x < width; x += 1) {
    const labelX = leftPad + x * cell + cell / 2;
    ctx.fillStyle = "#a48466";
    ctx.font = "12px 'Avenir Next', 'PingFang SC', sans-serif";
    ctx.textAlign = "center";
    ctx.fillText(String(x + 1), labelX, topPad - 16);
  }

  for (let y = 0; y < height; y += 1) {
    const labelY = topPad + y * cell + cell / 2 + 4;
    ctx.fillStyle = "#a48466";
    ctx.font = "12px 'Avenir Next', 'PingFang SC', sans-serif";
    ctx.textAlign = "right";
    ctx.fillText(String(y + 1), leftPad - 10, labelY);
  }

  ctx.textAlign = "center";
  for (let y = 0; y < height; y += 1) {
    for (let x = 0; x < width; x += 1) {
      const cellData = grid[y * width + x];
      const left = leftPad + x * cell;
      const top = topPad + y * cell;

      ctx.fillStyle = cellData ? cellData.hex : "#f3ede4";
      ctx.fillRect(left, top, cell, cell);

      ctx.strokeStyle = x % 5 === 0 || y % 5 === 0 ? "#8d6a4d" : "#cfbea9";
      ctx.lineWidth = x % 5 === 0 || y % 5 === 0 ? 1.3 : 0.7;
      ctx.strokeRect(left, top, cell, cell);

      if (!cellData) {
        continue;
      }

      const textColor = getReadableTextColor(cellData.rgb);
      ctx.fillStyle = textColor;
      ctx.font = `${Math.max(8, Math.floor(cell * 0.3))}px 'Avenir Next', 'PingFang SC', sans-serif`;
      ctx.fillText(cellData.code, left + cell / 2, top + cell / 2 + Math.max(3, cell * 0.1));
    }
  }

  ctx.textAlign = "left";
  const legendTop = topPad + gridHeight + 48;
  ctx.fillStyle = "#2f241a";
  ctx.font = "700 24px 'Avenir Next', 'PingFang SC', sans-serif";
  ctx.fillText("色号代码表 / 材料清单", leftPad, legendTop);

  ctx.fillStyle = "#6c523a";
  ctx.font = "15px 'Avenir Next', 'PingFang SC', sans-serif";
  ctx.fillText("每格代表 1 颗拼豆，代码采用 MARD 标准色号。", leftPad, legendTop + 24);

  legend.forEach((item, index) => {
    const column = Math.floor(index / legendRows);
    const row = index % legendRows;
    const itemLeft = leftPad + column * legendColumnWidth;
    const itemTop = legendTop + 46 + row * legendRowHeight;

    ctx.fillStyle = item.hex;
    ctx.fillRect(itemLeft, itemTop - 14, 24, 24);
    ctx.strokeStyle = "rgba(47,36,26,0.18)";
    ctx.lineWidth = 1;
    ctx.strokeRect(itemLeft, itemTop - 14, 24, 24);

    ctx.fillStyle = "#2f241a";
    ctx.font = "700 15px 'Avenir Next', 'PingFang SC', sans-serif";
    ctx.fillText(item.code, itemLeft + 36, itemTop + 2);

    ctx.fillStyle = "#6c523a";
    ctx.font = "14px 'Avenir Next', 'PingFang SC', sans-serif";
    ctx.fillText(`${item.hex} / ${item.count} 颗`, itemLeft + 88, itemTop + 2);
  });

  return canvas;
}

function handleLegendTableClick(event) {
  const button = event.target.closest("[data-action]");
  if (!button || !state.pattern) {
    return;
  }

  const action = button.dataset.action;
  const code = button.dataset.code;
  if (!code) {
    return;
  }

  if (action === "set-paint") {
    state.editor.paintCode = code;
    renderEditorControls();
    setStatus(`当前画笔色已切换为 ${code}。`);
    return;
  }

  if (action === "select-color") {
    selectCellsByCode(code);
    return;
  }

  if (action === "replace-all") {
    const row = button.closest("tr");
    const select = row?.querySelector(".legend-replace-select");
    const nextCode = select?.value;
    if (!nextCode) {
      return;
    }
    if (nextCode === code) {
      setStatus(`${code} 没有变化。`);
      return;
    }
    replaceCellsByCode(code, nextCode);
  }
}

function toggleSourceColorSampling() {
  if (!state.pattern || !state.image) {
    return;
  }
  state.editor.sampleSourceActive = !state.editor.sampleSourceActive;
  renderEditorControls();
  setStatus(state.editor.sampleSourceActive
    ? "原图吸色已开启，请点击原图需要参考的颜色位置。"
    : "已关闭原图吸色。");
}

function handleSourceCanvasClick(event) {
  if (!state.editor.sampleSourceActive || !state.image || !state.sourcePreviewRect) {
    return;
  }

  const rect = refs.sourceCanvas.getBoundingClientRect();
  const scaleX = refs.sourceCanvas.width / rect.width;
  const scaleY = refs.sourceCanvas.height / rect.height;
  const canvasX = (event.clientX - rect.left) * scaleX;
  const canvasY = (event.clientY - rect.top) * scaleY;
  const previewRect = state.sourcePreviewRect;

  if (
    canvasX < previewRect.x
    || canvasY < previewRect.y
    || canvasX > previewRect.x + previewRect.width
    || canvasY > previewRect.y + previewRect.height
  ) {
    setStatus("请点在原图图像区域内，不要点到留白。");
    return;
  }

  const imageX = Math.max(0, Math.min(
    state.image.naturalWidth - 1,
    Math.round(((canvasX - previewRect.x) / previewRect.width) * state.image.naturalWidth)
  ));
  const imageY = Math.max(0, Math.min(
    state.image.naturalHeight - 1,
    Math.round(((canvasY - previewRect.y) / previewRect.height) * state.image.naturalHeight)
  ));

  const sampled = sampleImagePixel(state.image, imageX, imageY);
  if (!sampled) {
    return;
  }

  const sampledLab = rgbToLab(sampled.r, sampled.g, sampled.b);
  const nextEntry = state.pattern.settings.protectDarkLines && isProtectedDarkPixel(sampledLab)
    ? findProtectedDarkMatch(sampledLab)
    : findBestPaletteMatch(sampledLab);

  state.editor.paintCode = nextEntry.code;
  state.editor.sampleSourceActive = false;
  renderEditorControls();
  setStatus(`已从原图吸色，当前画笔为 ${nextEntry.code} ${nextEntry.hex}。`);
}

function sampleImagePixel(image, x, y) {
  const sampleCanvas = document.createElement("canvas");
  sampleCanvas.width = image.naturalWidth;
  sampleCanvas.height = image.naturalHeight;
  const ctx = sampleCanvas.getContext("2d", { willReadFrequently: true });
  ctx.drawImage(image, 0, 0);
  const data = ctx.getImageData(x, y, 1, 1).data;
  if (!data || data.length < 4 || data[3] === 0) {
    return null;
  }
  return { r: data[0], g: data[1], b: data[2], a: data[3] };
}

function handleEditorPointerDown(event) {
  if (!state.pattern) {
    return;
  }

  const index = getGridIndexFromEvent(event);
  if (index == null) {
    return;
  }

  if (typeof refs.editorCanvas.setPointerCapture === "function" && event.pointerId != null) {
    refs.editorCanvas.setPointerCapture(event.pointerId);
  }

  state.editor.pointerDown = true;
  state.editor.lastIndex = index;
  state.editor.pointerAction = state.editor.mode;

  if (state.editor.mode === "paint") {
    state.editor.dirty = paintCell(index, state.editor.paintCode);
    renderEditor(state.pattern);
  } else {
    state.editor.selectionAnchor = index;
    state.editor.selectedIndices = getRectSelection(index, index);
    renderEditor(state.pattern);
    renderEditorControls();
  }
}

function handleEditorPointerMove(event) {
  if (!state.pattern || !state.editor.pointerDown) {
    return;
  }

  const index = getGridIndexFromEvent(event);
  if (index == null || index === state.editor.lastIndex) {
    return;
  }
  state.editor.lastIndex = index;

  if (state.editor.pointerAction === "paint") {
    const changed = paintCell(index, state.editor.paintCode);
    state.editor.dirty = state.editor.dirty || changed;
    if (changed) {
      renderEditor(state.pattern);
    }
    return;
  }

  if (state.editor.pointerAction === "select" && state.editor.selectionAnchor != null) {
    state.editor.selectedIndices = getRectSelection(state.editor.selectionAnchor, index);
    renderEditor(state.pattern);
    renderEditorControls();
  }
}

function handleEditorPointerLeave() {
  if (!state.pattern || !state.editor.pointerDown) {
    return;
  }
  if (state.editor.pointerAction === "paint" && state.editor.dirty) {
    refreshPatternViews("已更新画笔修改。");
    state.editor.dirty = false;
  }
}

function handleEditorPointerUp() {
  if (!state.pattern || !state.editor.pointerDown) {
    return;
  }

  state.editor.pointerDown = false;
  state.editor.pointerAction = null;
  state.editor.selectionAnchor = null;
  state.editor.lastIndex = null;

  if (state.editor.dirty) {
    refreshPatternViews("已更新画笔修改。");
    state.editor.dirty = false;
  } else {
    renderEditorControls();
  }
}

function getGridIndexFromEvent(event) {
  const pattern = state.pattern;
  const metrics = pattern?.editorMetrics;
  if (!pattern || !metrics) {
    return null;
  }

  const rect = refs.editorCanvas.getBoundingClientRect();
  const scaleX = refs.editorCanvas.width / rect.width;
  const scaleY = refs.editorCanvas.height / rect.height;
  const canvasX = (event.clientX - rect.left) * scaleX;
  const canvasY = (event.clientY - rect.top) * scaleY;

  const localX = canvasX - metrics.leftPad;
  const localY = canvasY - metrics.topPad;
  if (localX < 0 || localY < 0) {
    return null;
  }

  const x = Math.floor(localX / metrics.cell);
  const y = Math.floor(localY / metrics.cell);
  if (x < 0 || y < 0 || x >= pattern.width || y >= pattern.height) {
    return null;
  }

  return y * pattern.width + x;
}

function paintCell(index, code) {
  if (!state.pattern || !code) {
    return false;
  }
  const current = state.pattern.grid[index];
  const next = paletteByCode.get(code);
  if (!current || !next || current.code === next.code) {
    return false;
  }
  state.pattern.grid[index] = next;
  return true;
}

function getRectSelection(fromIndex, toIndex) {
  const pattern = state.pattern;
  if (!pattern) {
    return new Set();
  }

  const fromX = fromIndex % pattern.width;
  const fromY = Math.floor(fromIndex / pattern.width);
  const toX = toIndex % pattern.width;
  const toY = Math.floor(toIndex / pattern.width);

  const minX = Math.min(fromX, toX);
  const maxX = Math.max(fromX, toX);
  const minY = Math.min(fromY, toY);
  const maxY = Math.max(fromY, toY);
  const selected = new Set();

  for (let y = minY; y <= maxY; y += 1) {
    for (let x = minX; x <= maxX; x += 1) {
      const index = y * pattern.width + x;
      if (pattern.grid[index]) {
        selected.add(index);
      }
    }
  }

  return selected;
}

function applySelectedCells() {
  if (!state.pattern || !state.editor.selectedIndices.size) {
    return;
  }

  let changed = 0;
  for (const index of state.editor.selectedIndices) {
    if (paintCell(index, state.editor.paintCode)) {
      changed += 1;
    }
  }

  if (!changed) {
    setStatus("选中格没有发生颜色变化。");
    return;
  }

  refreshPatternViews(`已批量修改 ${changed} 个格子。`);
}

function clearSelection() {
  if (!state.pattern) {
    return;
  }
  state.editor.selectedIndices = new Set();
  renderEditor(state.pattern);
  renderEditorControls();
  setStatus("已清空选择。");
}

function selectCellsByPaintColor() {
  if (!state.pattern || !state.editor.paintCode) {
    return;
  }
  selectCellsByCode(state.editor.paintCode);
}

function selectCellsByCode(code) {
  if (!state.pattern) {
    return;
  }

  const selected = new Set();
  state.pattern.grid.forEach((cell, index) => {
    if (cell?.code === code) {
      selected.add(index);
    }
  });
  state.editor.selectedIndices = selected;
  renderEditor(state.pattern);
  renderEditorControls();
  setStatus(`已选中 ${code} 色号的 ${selected.size} 个格子。`);
}

function replaceCellsByCode(fromCode, toCode) {
  if (!state.pattern) {
    return;
  }

  const next = paletteByCode.get(toCode);
  if (!next) {
    return;
  }

  let changed = 0;
  state.pattern.grid = state.pattern.grid.map((cell) => {
    if (cell?.code !== fromCode) {
      return cell;
    }
    changed += 1;
    return next;
  });

  state.editor.paintCode = toCode;
  state.editor.selectedIndices = new Set();
  refreshPatternViews(`已将 ${fromCode} 整体替换为 ${toCode}，共 ${changed} 格。`);
}

function buildPaletteOptionsMarkup(selectedCode) {
  return palette.map((entry) => `
    <option value="${entry.code}" ${entry.code === selectedCode ? "selected" : ""}>
      ${entry.code} ${entry.hex}
    </option>
  `).join("");
}

function drawImagePreview(image) {
  drawContainImage(refs.sourceCanvas, image);
}

function renderEmptyCanvases() {
  state.sourcePreviewRect = null;
  drawCanvasPlaceholder(refs.sourceCanvas, "等待上传图片");
  drawCanvasPlaceholder(refs.patternCanvas, "等待生成图纸");
}

function renderEmptyEditor() {
  drawCanvasPlaceholder(refs.editorCanvas, "生成图纸后可在这里编辑");
  renderEditorControls();
}

function drawCanvasPlaceholder(canvas, text) {
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = "#fff";
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = "#b59d84";
  ctx.font = "700 28px 'Avenir Next', 'PingFang SC', sans-serif";
  ctx.textAlign = "center";
  ctx.fillText(text, canvas.width / 2, canvas.height / 2);
}

function drawContainImage(canvas, image) {
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = "#fff";
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  const rect = getDrawRect(image.naturalWidth, image.naturalHeight, canvas.width, canvas.height, "contain");
  ctx.drawImage(image, rect.x, rect.y, rect.width, rect.height);
  if (canvas === refs.sourceCanvas) {
    state.sourcePreviewRect = rect;
  }
}

function downloadPng() {
  if (!state.pattern) {
    return;
  }
  const link = document.createElement("a");
  link.href = state.pattern.rendered.toDataURL("image/png");
  link.download = `pindou-pattern-${state.pattern.width}x${state.pattern.height}.png`;
  link.click();
}

function downloadCsv() {
  if (!state.pattern) {
    return;
  }
  const header = ["code", "hex", "count", "ratio"];
  const rows = state.pattern.legend.map((item) => [
    item.code,
    item.hex,
    item.count,
    `${((item.count / state.pattern.totalBeads) * 100).toFixed(2)}%`,
  ]);
  const csv = [header, ...rows].map((row) => row.join(",")).join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = `pindou-materials-${state.pattern.width}x${state.pattern.height}.csv`;
  link.click();
}

function setStatus(text) {
  refs.statusText.textContent = text;
}

function statItem(label, value) {
  return `<div><dt>${label}</dt><dd>${value}</dd></div>`;
}

function clampInt(value, min, max, fallback) {
  const number = Number.parseInt(value, 10);
  if (Number.isNaN(number)) {
    return fallback;
  }
  return Math.min(max, Math.max(min, number));
}

function getDrawRect(sourceWidth, sourceHeight, targetWidth, targetHeight, mode) {
  const sourceRatio = sourceWidth / sourceHeight;
  const targetRatio = targetWidth / targetHeight;
  const useCover = mode === "cover";
  let width;
  let height;

  if ((useCover && sourceRatio > targetRatio) || (!useCover && sourceRatio < targetRatio)) {
    height = targetHeight;
    width = height * sourceRatio;
  } else {
    width = targetWidth;
    height = width / sourceRatio;
  }

  return {
    width,
    height,
    x: (targetWidth - width) / 2,
    y: (targetHeight - height) / 2,
  };
}

function nearestLabIndex(target, labs) {
  let bestIndex = 0;
  let bestDistance = Infinity;
  for (let index = 0; index < labs.length; index += 1) {
    const distance = labDistance(target, labs[index]);
    if (distance < bestDistance) {
      bestDistance = distance;
      bestIndex = index;
    }
  }
  return bestIndex;
}

function nearestLabDistance(target, labs) {
  return labs.reduce((best, lab) => Math.min(best, labDistance(target, lab)), Infinity);
}

function labDistance(left, right) {
  const dl = left.l - right.l;
  const da = left.a - right.a;
  const db = left.b - right.b;
  return dl * dl + da * da + db * db;
}

function deltaE2000(left, right) {
  const avgLp = (left.l + right.l) / 2;
  const c1 = Math.sqrt((left.a ** 2) + (left.b ** 2));
  const c2 = Math.sqrt((right.a ** 2) + (right.b ** 2));
  const avgC = (c1 + c2) / 2;
  const g = 0.5 * (1 - Math.sqrt((avgC ** 7) / ((avgC ** 7) + (25 ** 7))));

  const a1p = (1 + g) * left.a;
  const a2p = (1 + g) * right.a;
  const c1p = Math.sqrt((a1p ** 2) + (left.b ** 2));
  const c2p = Math.sqrt((a2p ** 2) + (right.b ** 2));
  const avgCp = (c1p + c2p) / 2;

  const h1p = normalizeHueRadians(Math.atan2(left.b, a1p));
  const h2p = normalizeHueRadians(Math.atan2(right.b, a2p));

  const deltaLp = right.l - left.l;
  const deltaCp = c2p - c1p;

  let deltahp = 0;
  if (c1p * c2p !== 0) {
    if (Math.abs(h2p - h1p) <= Math.PI) {
      deltahp = h2p - h1p;
    } else if (h2p <= h1p) {
      deltahp = (h2p - h1p) + (2 * Math.PI);
    } else {
      deltahp = (h2p - h1p) - (2 * Math.PI);
    }
  }

  const deltaHp = 2 * Math.sqrt(c1p * c2p) * Math.sin(deltahp / 2);

  let avgHp = h1p + h2p;
  if (c1p * c2p === 0) {
    avgHp = h1p + h2p;
  } else if (Math.abs(h1p - h2p) > Math.PI) {
    avgHp = (h1p + h2p + (2 * Math.PI)) / 2;
  } else {
    avgHp = (h1p + h2p) / 2;
  }

  const t = 1
    - (0.17 * Math.cos(avgHp - degToRad(30)))
    + (0.24 * Math.cos(2 * avgHp))
    + (0.32 * Math.cos((3 * avgHp) + degToRad(6)))
    - (0.20 * Math.cos((4 * avgHp) - degToRad(63)));

  const deltaTheta = degToRad(30) * Math.exp(-(((radToDeg(avgHp) - 275) / 25) ** 2));
  const rc = 2 * Math.sqrt((avgCp ** 7) / ((avgCp ** 7) + (25 ** 7)));
  const sl = 1 + ((0.015 * ((avgLp - 50) ** 2)) / Math.sqrt(20 + ((avgLp - 50) ** 2)));
  const sc = 1 + (0.045 * avgCp);
  const sh = 1 + (0.015 * avgCp * t);
  const rt = -(Math.sin(2 * deltaTheta) * rc);

  return Math.sqrt(
    (deltaLp / sl) ** 2
    + (deltaCp / sc) ** 2
    + (deltaHp / sh) ** 2
    + (rt * (deltaCp / sc) * (deltaHp / sh))
  );
}

function normalizeHueRadians(value) {
  return value >= 0 ? value : value + (2 * Math.PI);
}

function degToRad(value) {
  return value * (Math.PI / 180);
}

function radToDeg(value) {
  return value * (180 / Math.PI);
}

function getReadableTextColor(rgb) {
  const luminance = (0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b) / 255;
  return luminance > 0.62 ? "#2f241a" : "#fffaf4";
}

function hexToRgb(hex) {
  const normalized = hex.replace("#", "");
  return {
    r: Number.parseInt(normalized.slice(0, 2), 16),
    g: Number.parseInt(normalized.slice(2, 4), 16),
    b: Number.parseInt(normalized.slice(4, 6), 16),
  };
}

function rgbToLab(r, g, b) {
  let red = r / 255;
  let green = g / 255;
  let blue = b / 255;

  red = red > 0.04045 ? ((red + 0.055) / 1.055) ** 2.4 : red / 12.92;
  green = green > 0.04045 ? ((green + 0.055) / 1.055) ** 2.4 : green / 12.92;
  blue = blue > 0.04045 ? ((blue + 0.055) / 1.055) ** 2.4 : blue / 12.92;

  const x = (red * 0.4124 + green * 0.3576 + blue * 0.1805) / 0.95047;
  const y = red * 0.2126 + green * 0.7152 + blue * 0.0722;
  const z = (red * 0.0193 + green * 0.1192 + blue * 0.9505) / 1.08883;

  const fx = x > 0.008856 ? x ** (1 / 3) : (7.787 * x) + (16 / 116);
  const fy = y > 0.008856 ? y ** (1 / 3) : (7.787 * y) + (16 / 116);
  const fz = z > 0.008856 ? z ** (1 / 3) : (7.787 * z) + (16 / 116);

  return {
    l: (116 * fy) - 16,
    a: 500 * (fx - fy),
    b: 200 * (fy - fz),
  };
}
