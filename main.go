package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syblog/config"
	"syblog/logger"
	"syblog/render"
	"syblog/service"
	"time"

	"github.com/88250/lute"
	"github.com/88250/lute/parse"
	"github.com/88250/lute/util"
	"github.com/dustin/go-humanize"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var publishMap = make(map[string]struct{})

func main() {
	logger.Info("读取配置")
	logger.Infof("思源笔记API地址：%s", config.GetConfig().SY.APIURL)
	logger.Infof("工作空间路径：%s", config.GetConfig().SY.WorkspacePath)
	hugoContentPath := filepath.Join(config.GetConfig().Hugo.BlogPath, "content", config.GetConfig().Hugo.SectionName)
	logger.Infof("清理Hugo目录：%s", hugoContentPath)
	os.RemoveAll(hugoContentPath)
	logger.Info("获取需要发布的文章列表")
	articles := service.FindArticleList()
	logger.Infof("需要发布的文章数（直接）：%d", articles.Len())
	logger.Info("开始逐个发布文章")
	for e := articles.Front(); e != nil; e = e.Next() {
		article := e.Value.(*service.Article)
		if _, ok := publishMap[article.ID]; ok {
			continue
		}
		logger.Infof("开始发布：%s", article.Title)
		md, err := service.ExportMD(article.ID)
		if err != nil {
			logger.Errorf("%+v", errors.WithStack(err))
			continue
		}
		luteEngine := lute.New()
		tree := parse.Parse("", []byte(md), luteEngine.ParseOptions)
		luteEngine.RenderOptions.AutoSpace = true
		luteEngine.RenderOptions.FixTermTypo = true
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
	cmd := exec.Command(config.GetConfig().Hugo.ExcutePath)
	cmd.Dir = config.GetConfig().Hugo.BlogPath
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}

	if config.GetConfig().SSH.Addr == "" {
		return
	}
	logger.Info("打包压缩站点")
	tempFilePath := packageSite()
	defer os.RemoveAll(filepath.Join(tempFilePath, "../"))

	logger.Info("连接服务器SFTP")
	var conf *ssh.ClientConfig
	if config.GetConfig().SSH.Password != "" {
		conf = &ssh.ClientConfig{
			User: "ubuntu",
			Auth: []ssh.AuthMethod{
				ssh.Password(config.GetConfig().SSH.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else if config.GetConfig().SSH.KeyPath != "" {
		key, err := ioutil.ReadFile(config.GetConfig().SSH.KeyPath)
		if err != nil {
			logger.Fatalf("%+v", errors.WithStack(err))
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			logger.Fatalf("%+v", errors.WithStack(err))
		}
		conf = &ssh.ClientConfig{
			User: "ubuntu",
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else {
		logger.Fatal("SSH的密码或私钥文件路径未配置")
	}

	client, err := ssh.Dial("tcp", config.GetConfig().SSH.Addr, conf)
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	defer sftpClient.Close()
	logger.Info("上传站点压缩包")
	target, err := sftpClient.OpenFile("/tmp/site.tar.gz", os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	defer target.Close()
	src, err := os.Open(tempFilePath)
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	defer src.Close()
	stat, _ := src.Stat()
	logger.Infof("站点压缩包大小为：%s", humanize.Bytes(uint64(stat.Size())))
	_, err = io.Copy(target, src)
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}

	logger.Info("解压至指定路径")
	session, err := client.NewSession()
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	defer session.Close()

	rmPath := config.GetConfig().SSH.SitePath
	rmPath = strings.TrimSuffix(rmPath, "/")
	rmPath = rmPath + "/*"
	err = session.Run(fmt.Sprintf("rm -rf %s && tar -zxf /tmp/site.tar.gz -C %s", rmPath, config.GetConfig().SSH.SitePath))
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	logger.Info("执行完成")
}

func packageSite() string {
	tempDir, err := ioutil.TempDir("", "sitezip-*")
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	tempFile, err := ioutil.TempFile(tempDir, "*.tar.gz")
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	defer tempFile.Close()
	gw := gzip.NewWriter(tempFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	publicPath := filepath.Join(config.GetConfig().Hugo.BlogPath, "public")
	fis, err := ioutil.ReadDir(publicPath)
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	for _, fi := range fis {
		p := filepath.Join(publicPath, fi.Name())
		f, err := os.Open(p)
		if err != nil {
			logger.Fatalf("%+v", errors.WithStack(err))
		}
		err = compress(f, "", tw)
		if err != nil {
			logger.Fatalf("%+v", errors.WithStack(err))
		}
	}
	return tempFile.Name()
}

func exportArticle(article *service.Article) {
	fmMap := make(map[string]any)
	fmMap["title"] = article.Title
	fmMap["date"] = tomlLocalDateTime(article.Created)
	fmMap["lastmod"] = tomlLocalDateTime(article.Updated)
	fmMap["tags"] = article.Tags
	attrs := service.FindAttrs(article.ID)
	for k, v := range attrs {
		if k == "date" || k == "lastmod" {
			fmMap[k] = tomlLocalDateTime(v.(time.Time))
		} else {
			fmMap[k] = v
		}
	}
	frontMatter, err := toml.Marshal(fmMap)
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, ""))
	}
	articleDirPath := filepath.Join(config.GetConfig().Hugo.BlogPath, "content", config.GetConfig().Hugo.SectionName, article.Title)
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

	links := service.FindLinkTo(article.ID)
	if len(links) > 0 {
		file.WriteString("\r\n\r\n---\r\n\r\n反链：\r\n\r\n")
		for i, l := range links {
			file.WriteString(strconv.Itoa(i + 1))
			file.WriteString(". ")
			file.WriteString("[")
			file.WriteString(l[0])
			file.WriteString("](")
			file.WriteString(l[1])
			file.WriteString(")\r\n")
		}
	}

	// 输出资源文件
	assertDirPath := filepath.Join(articleDirPath, "assets")
	isFirst := true
	for _, a := range article.Asserts {
		p := filepath.Join(config.GetConfig().SY.AssetsPath, a)
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

func compress(file *os.File, prefix string, tw *tar.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, tw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := tar.FileInfoHeader(info, "")
		header.Name = prefix + "/" + header.Name
		header.Name = strings.TrimPrefix(header.Name, "/")
		logger.Infof("正在压缩：%s", header.Name)
		if err != nil {
			return err
		}
		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, file)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func tomlLocalDateTime(t time.Time) toml.LocalDateTime {
	return toml.LocalDateTime{
		LocalDate: toml.LocalDate{
			Year:  t.Year(),
			Month: int(t.Month()),
			Day:   t.Day(),
		},
		LocalTime: toml.LocalTime{
			Hour:   t.Hour(),
			Minute: t.Minute(),
			Second: t.Second(),
		},
	}
}
