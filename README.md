# MIRRORS-OS

## 整一个自己的镜像站！

> 在此对 [kaiakz/rsync-os](https://github.com/kaiakz/rsync-os) 以及 [K265/teambition-pan-api](https://github.com/K265/teambition-pan-api) 表示感谢

# 简介

利用本项目你可以从传统的镜像站使用 `rsync` 协议同步文件到一些储存后端并提供一个可以将用户请求正确定向到储存后端直链的 `http` 服务。

在这种情况下只需要很小的 网络/磁盘 成本就可以自建一个镜像站！

> NOTE  
> 1. 服务器只负责从上游下载文件并上传到存储后端，并不负责文件的分发。  
> 2. 同步过程中从上游下载的文件大部分不会落盘，只有单个文件过大时才会暂存在服务器磁盘上。  


# 使用

## 1. 编辑配置文件

> !!! update 配置文件格式已经更新，请自行通过样例配置查阅新增字段，以下配置过时

配置文件样例如下


```toml
title = "configuration of mirrors-os"

[global]
  server = "127.0.0.1:52111"

[archlinux]
  src = "rsync://mirrors.tuna.tsinghua.edu.cn/archlinux/pool/community"
  cookie = "TEAMBITION_SESSIONID=xxx; TEAMBITION_SESSIONID.sig=xxx"
  base = "mirrors/"
  dbpath = "archlinux.db"
  cron = "0 00 21 * * *"

[archlinuxcn]
  src = "rsync://mirrors.tuna.tsinghua.edu.cn/archlinuxcn/"
  cookie = "TEAMBITION_SESSIONID=xxx; TEAMBITION_SESSIONID.sig=xxx"
  base = "mirrors/"
  dbpath = "archlinuxcn.db"
  cron = "0 07 21 * * *"
```

在 `[global]` 一节 `server` 字段指定服务器地址

编写其余小节配置同步任务，以 `[archlinux]` 一节为例`：  

- `archlinux` 为任务名
- `src` 字段指定上游地址
> 在 `src` 中你可以指定任意子路径，例如 `rsync://xxx.xx/archlinux/pool/packages` 是可以的
- `cookie` 字段填写 `teambition` 的 cookie
- `base` 字段为镜像在 `teambition` 的路径
- `dbpath` 为同步数据库所在的路径
- `cron` 设置定时任务，注意第一位为秒

### 注意！

你的数据库文件非常重要，请保持它与 `teambition` 的同步。为此你需要做到：
- 让 `mirrors-os` 作为 `teambition` 的 `base` 路径的唯一写操作者
- 不要让 `mirrors-os` 异常退出（比如突然断电），使用 `Ctrl-C` 杀死它是被允许的

## 运行

```shell
mirrors-os [tasknames...]
```

# 关于访问目录
访问目录时的 html 页面是自动生成的，在程序启动和同步成功时会重新生成（同步中页面显示会和实际不一致）。

访问文件不存在延迟问题。

# 初始同步的建议

初始同步需要改动的文件会非常多，可以把一个镜像任务分成若干子任务 **逐个** 完成。

分成若干任务的方式为，使用子路径作为 `src`
例如可以把 `archlinux` 的初始同步任务分成三次， `src` 依次为

`rsync://xxx.xx/archlinux/pool/packages`

`rsync://xxx.xx/archlinux/pool/community`

`rsync://xxx.xx/archlinux/`

注意三次任务应使用同步一个数据库文件.



（反正坑挺多的，多踩踩就放弃使用了
