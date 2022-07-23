package service

import (
	"container/list"
	"fmt"
	"strings"
	"syblog/logger"
	"time"

	"github.com/imroc/req/v3"
	"github.com/pkg/errors"
)

var client = req.C()

type Article struct {
	Title   string    `toml:"title"`
	Created time.Time `toml:"date"`
	Updated time.Time `toml:"lastmod"`
	Tags    []string  `toml:"tags"`
	Content string    `toml:"-"`
	Linked  []string  `toml:"-"`
	Asserts []string  `toml:"-"`
	ID      string    `toml:"-"`
}

func FindArticleList() *list.List {
	l, err := findList("select * from blocks where type='d' and id in (select block_id from attributes where name='custom-publish' and value='1')")
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	as := list.New()
	for _, doc := range l {
		article := &Article{}
		article.ID = doc["id"].(string)
		article.Title = doc["content"].(string)
		createdStr := doc["created"].(string)
		article.Created, err = time.Parse("20060102150405", createdStr)
		if err != nil {
			logger.Fatalf("%+v", errors.WithStack(err))
		}
		updatedStr := doc["updated"].(string)
		if updatedStr == "" {
			updatedStr = createdStr
		}
		article.Updated, err = time.Parse("20060102150405", updatedStr)
		if err != nil {
			logger.Fatalf("%+v", errors.WithStack(err))
		}
		tagStr := doc["tag"].(string)
		if tagStr == "" {
			article.Tags = []string{}
		} else {
			tags := strings.Split(tagStr, " ")
			for i := 0; i < len(tags); i++ {
				tags[i] = strings.Trim(tags[i], "#")
			}
			article.Tags = tags
		}
		as.PushBack(article)
	}
	return as
}

func FindArticleByBlockID(blockID string) *Article {
	l, err := findList(fmt.Sprintf("select * from blocks where type='d' and id = (select root_id from blocks where id='%s')", blockID))
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	if len(l) != 1 {
		logger.Errorf("未找到对应文档，blockID为：%s", blockID)
		return nil
	}
	doc := l[0]
	article := &Article{}
	article.ID = doc["id"].(string)
	article.Title = doc["content"].(string)
	createdStr := doc["created"].(string)
	article.Created, err = time.Parse("20060102150405", createdStr)
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	updatedStr := doc["updated"].(string)
	if updatedStr == "" {
		updatedStr = createdStr
	}
	article.Updated, err = time.Parse("20060102150405", updatedStr)
	if err != nil {
		logger.Errorf("%+v", errors.WithStack(err))
		article.Updated = article.Created
	}
	tagStr := doc["tag"].(string)
	tags := strings.Split(tagStr, " ")
	for i := 0; i < len(tags); i++ {
		tags[i] = strings.Trim(tags[i], "#")
	}
	article.Tags = tags
	return article
}

func ExportMD(id string) (string, error) {
	result := &struct {
		Result
		Data map[string]string `json:"data"`
	}{}
	_, err := client.R().SetBody(map[string]interface{}{
		"id": id,
	}).SetResult(result).Post("http://127.0.0.1:6806/api/export/exportMdContent")
	if err != nil {
		return "", err
	}
	return result.Data["content"], nil
}

func findList(sql string) ([]map[string]any, error) {
	result := &struct {
		Result
		Data []map[string]interface{} `json:"data"`
	}{}
	_, err := client.R().
		SetBody(map[string]interface{}{
			"stmt": sql,
		}).SetResult(result).
		Post("http://127.0.0.1:6806/api/query/sql")
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func init() {
	client.OnBeforeRequest(func(c *req.Client, r *req.Request) error {
		r.SetHeader("Authorization", "Token 78u8eihczoo8eb8s")
		return nil
	})
}
