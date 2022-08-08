package service

import (
	"container/list"
	"fmt"
	"net/url"
	"strings"
	"syblog/config"
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
	}).SetResult(result).Post("http://" + config.GetConfig().SY.APIURL + "/api/export/exportMdContent")
	if err != nil {
		return "", err
	}
	return result.Data["content"], nil
}

func FindAttrs(id string) map[string]any {
	attrs, err := findList("select name,value from attributes where root_id='" + id + "' and name like 'custom-sn-%'")
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	ret := make(map[string]any)
	for _, attr := range attrs {
		key := attr["name"].(string)
		key = strings.TrimPrefix(key, "custom-sn-")
		value := attr["value"].(string)
		if key == "date" || key == "lastmod" {
			d, err := time.Parse("2006-01-02T15:04:05", value)
			if err != nil {
				logger.Errorf("%+v", errors.WithStack(err))
				continue
			}
			ret[key] = d
		} else {
			ret[key] = value
		}
	}
	return ret
}

func FindLinkTo(id string) [][2]string {
	ids, err := findList("select root_id from refs where def_block_root_id='" + id + "'")
	if err != nil {
		logger.Fatalf("%+v", errors.WithStack(err))
	}
	ret := make([][2]string, 0)
	for _, d := range ids {
		aid := d["root_id"].(string)
		ls, err := findList("select content from blocks where id='" + aid + "'")
		if err != nil {
			logger.Errorf("%+v", errors.WithStack(err))
			continue
		}
		if len(ls) == 0 {
			logger.Errorf("没有找到对应的文档块，id为：%s", aid)
			continue
		}
		title := ls[0]["content"].(string)
		item := [2]string{}
		item[0] = title
		t := strings.ReplaceAll(title, " ", "-")
		link := "../" + url.QueryEscape(strings.ToLower(t)) + "/"
		item[1] = link
		ret = append(ret, item)
	}
	return ret
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
		Post("http://" + config.GetConfig().SY.APIURL + "/api/query/sql")
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
		r.SetHeader("Authorization", "Token "+config.GetConfig().SY.APIToken)
		return nil
	})
}
