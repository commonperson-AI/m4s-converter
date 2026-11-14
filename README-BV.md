# 按 BV 号选择转换（Windows 缓存版）

这是 mzky/m4s-converter 的定制分支，保留原项目许可证。

## 双击使用

下载 Windows ZIP，全部解压到同一目录，双击 `select-bv.cmd`。
默认缓存目录是 `D:\BaiduNetdiskDownload\bilibili`。启动器会调用旁边的新程序
`m4s-converter-bv.exe`，不覆盖 Downloads 中的旧版本，也不依赖旧快捷方式。
按转换器提示继续，然后输入一个或多个完整 BV 号或视频链接，空格或逗号分隔。
直接回车取消。默认不下载或生成弹幕。

同一 BV 号的所有已缓存分 P 都会被选中。若任何输入 BV 没有完整缓存，整个任务在转换前停止，
请修正输入后重试。按 BV 号匹配大小写，不按文件夹名猜测。

输出：`缓存目录\output\BV号\P序号_缓存文件夹名_标题.mp4`。
原始 `.m4s` 不删除；与上游一样，选中缓存目录会生成中间音视频文件。
未选中的缓存只读取元数据，不转换。重复运行保留已有同名成品，不覆盖。
播放确认成品后再上传 Google 相册。

## 命令行

```bat
m4s-converter-bv.exe -c "D:\BaiduNetdiskDownload\bilibili" --select -a
m4s-converter-bv.exe -c "D:\BaiduNetdiskDownload\bilibili" --bv "BV1CpR1BiEuf BV1234567890" -a
m4s-converter-bv.exe -c "D:\BaiduNetdiskDownload\bilibili" --list
m4s-converter-bv.exe -c "D:\BaiduNetdiskDownload\bilibili" --bv "BV1CpR1BiEuf" --output-dir "D:\Videos\Bilibili" -a
```

`--list` 只列出条目。`--select` 先列出缓存再读取选择。
新筛选功能针对 `videoInfo.json` / `.videoInfo` 与两个 `.m4s` 同目录的 Windows 缓存。
Android 缓存保留上游处理方式，尚未接入 BV 筛选。
新 EXE 默认进入 BV 选择模式；Windows 默认缓存路径同上。只有明确提供 `--all` 才会全量转换。
本分支关闭了上游版本更新提示，避免误装不含筛选功能的原版。

## 构建和验证

Go 1.23 或更高：

```sh
go test ./...
go vet ./...
GOOS=windows GOARCH=amd64 go build -trimpath -o m4s-converter-bv.exe .
```

回归测试覆盖 BV/链接输入、空输入取消、元数据去重、分 P、缺失与未完成缓存、
输出目录排除、未选中文件不处理，以及移除缓存前缀时的数据完整性。
