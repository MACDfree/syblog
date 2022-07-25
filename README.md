# 思源笔记&Hugo博客发布工具

SYBlog是一个将思源笔记通过Hugo发布成博客的工具。

## 快速开始

首先，对思源笔记中需要发布的文档设置属性，属性名称为publish，值为1。

然后，修改config.toml配置文件。配置描述如下：

```toml
[siyuan]
apiURL = ""        # 思源笔记API地址，默认为：127.0.0.1:6806
workspacePath = "" # 使用的工作空间路径，如：D:\\synote

[hugo]
excutePath = ""  # hugo可执行程序路径，如：D:\\software\\bin\\hugo.exe
blogPath = ""    # 博客路径，如：D:\\code\\hugoblog
sectionName = "" # 生成的section名字，默认为notes

[ssh]
addr = ""     # 自己的VPS服务器地址，如：231.21.21.21
user = ""     # 登录账号，如：root
password = "" # 登录密码（与keyPath二选一），如：123456
keyPath = ""  # 登录使用的私钥（与password二选一），如：D:\\privatekey\\id_rsa
sitePath = "" # VPS服务器上站点路径，如：/home/user/nginx/www
```

最后，双击执行syblog.exe即可。

## 功能描述

SYBlog的功能为：

1. 通过思源笔记查询SQL的API获取需要发布的文档信息；
2. 通过思源笔记导出Markdown的API获取文档的Markdown内容；
3. 通过lute解析Markdown，获取需要复制的图片等资源，并且将引用块改成普通链接，同时将被引用的文档也进行导出；
4. 将Markdown写入hugo博客中指定文件夹；
5. 调用hugo命令生成静态页面；
6. 将生成好的静态页面打包；
7. 上传打包文件至远程服务器；
8. 在远程服务器上执行解压缩命令，完成发布。
