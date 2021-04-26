# 4g-gateway

在 树莓派 和 imx6ull 上测试通过

需要使用 go1.13 以上版本编译
只支持在 Mac，Linux 系统上编译
只能在以下平台运行
darwin/amd64, darwin/arm64, darwin/arm,
linux/amd64, linux/386,
linux/arm64, linux/arm,
linux/mips, linux/mips64, linux/mipsle, linux/mips64le

## 编译

在 mac 系统中运行

```sh
make mac
```

在 ARM 32 位系统中运行

```sh
make arm32
```

在 ARM 64 位系统中运行

```sh
make arm64
```

## Sqlite 数据库

| 表名 | 字段  | 类型   | 含义     | 备注        |
| ---- | ----- | ------ | -------- | ----------- |
| mqtt | topic | string | 消息类型 | GPS、Status |
| mqtt | msg   | string | 消息体   |             |
