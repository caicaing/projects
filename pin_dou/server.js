const http = require("node:http");
const fs = require("node:fs");
const path = require("node:path");

const root = __dirname;
const host = "127.0.0.1";
const port = Number.parseInt(process.env.PORT || "4173", 10);

const mimeTypes = {
  ".css": "text/css; charset=utf-8",
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".png": "image/png",
  ".svg": "image/svg+xml",
};

const server = http.createServer((request, response) => {
  const requestUrl = request.url === "/" ? "/index.html" : request.url;
  const safePath = path.normalize(requestUrl).replace(/^(\.\.[/\\])+/, "");
  const filePath = path.join(root, safePath);

  fs.readFile(filePath, (error, content) => {
    if (error) {
      response.statusCode = error.code === "ENOENT" ? 404 : 500;
      response.setHeader("Content-Type", "text/plain; charset=utf-8");
      response.end(error.code === "ENOENT" ? "Not found" : "Server error");
      return;
    }

    response.statusCode = 200;
    response.setHeader("Content-Type", mimeTypes[path.extname(filePath)] || "application/octet-stream");
    response.end(content);
  });
});

server.listen(port, host, () => {
  console.log(`拼豆图纸工具已启动: http://${host}:${port}`);
});
