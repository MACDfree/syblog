package main

import (
	"io"
	"os"
	"path/filepath"
	"syblog/config"
	"syblog/logger"
	"syblog/render"
	"syblog/service"

	"github.com/88250/lute"
	"github.com/88250/lute/parse"
	"github.com/88250/lute/util"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

var publishMap = make(map[string]struct{})

func main() {
	logger.Info("读取配置")
	logger.Infof("思源笔记API地址：%s", config.GetConfig().SiYuanAPI)
	logger.Infof("工作空间路径：%s", config.GetConfig().WorkspacePath)
	logger.Infof("生成到Hugo的路径：%s", config.GetConfig().HugoGenPath)
	logger.Info("获取需要发布的文章列表")
	articles := service.FindArticleList()
	logger.Infof("需要发布的文章数（直接）：%d", articles.Len())
	logger.Info("开始逐个发布文章")
	for e := articles.Front(); e != nil; e = e.Next() {
		article := e.Value.(*service.Article)
		logger.Infof("开始发布：%s", article.Title)
		md, err := service.ExportMD(article.ID)
		if err != nil {
			logger.Errorf("%+v", errors.WithStack(err))
			continue
		}
		luteEngine := lute.New()
		tree := parse.Parse("demo", []byte(md), luteEngine.ParseOptions)
		renderer := render.NewFormatRenderer(tree, luteEngine.RenderOptions, article, articles, publishMap)
		formattedBytes := renderer.Render()
		md = util.BytesToStr(formattedBytes)
		article.Content = md
		exportArticle(article)
		publishMap[article.ID] = struct{}{}
		logger.Infof("完成发布：%s", article.Title)
	}
	logger.Infof("总共发布的文章数：%d", articles.Len())

	logger.Info("执行Hugo生成站点")
	logger.Info("打包压缩站点")
	logger.Info("连接服务器SFTP")
	logger.Info("上传站点压缩包")
	logger.Info("解压至指定路径")
}

func exportArticle(article *service.Article) {
	frontMatter, err := toml.Marshal(article)
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, ""))
	}
	articleDirPath := filepath.Join(config.GetConfig().HugoGenPath, article.Title)
	if _, err := os.Stat(articleDirPath); err != nil {
		os.MkdirAll(articleDirPath, 0555)
	}
	mdFilePath := filepath.Join(articleDirPath, "index.md")
	file, err := os.Create(mdFilePath)
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, ""))
	}
	defer file.Close()
	file.Write([]byte("+++\r\n"))
	file.Write(frontMatter)
	file.Write([]byte("+++\r\n\r\n"))

	file.WriteString(article.Content)

	// 输出资源文件
	assertDirPath := filepath.Join(articleDirPath, "assets")
	isFirst := true
	for _, a := range article.Asserts {
		p := filepath.Join(config.GetConfig().AssetsPath, a)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		src, err := os.Open(p)
		if err != nil {
			logger.Fatalf("%+v", errors.Wrap(err, ""))
		}

		if isFirst {
			if _, err := os.Stat(assertDirPath); err != nil {
				os.MkdirAll(assertDirPath, 0555)
			} else {
				os.RemoveAll(assertDirPath)
				os.Mkdir(assertDirPath, 0555)
			}
			isFirst = false
		}

		p = filepath.Join(assertDirPath, a)
		dst, err := os.Create(p)
		if err != nil {
			logger.Fatalf("%+v", errors.Wrap(err, ""))
		}

		io.Copy(dst, src)
	}
}
