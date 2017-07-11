gorouter
==========

从CloudFoundry gorouter(tag 45ca951297) fork，更改实现。

主要修改点：
CloudFoundry gorouter通过侦听NATS汇报的信息生成路由表；
而现在gorouter去redis里读取相应信息生成路由表（gorouter会在内存中保存份路由表，如果redis宕掉将暂停更新路由表）。

router启动时，从redis中加载路由表（URL与rs\_ip:port的对应关系，以及CNAME与URL的对应关系），格式如

```
redis 127.0.0.1:6379> keys *
1) "/rs/demo.xae.xiaomi.com"
3) "/rs/test.xae.xiaomi.com"
4) "/cname/ulricqin.com"
6) "/rs/api2.xae.xiaomi.com"
redis 127.0.0.1:6379> lrange /rs/demo.xae.xiaomi.com 0 -1
1) "10.201.37.5:10005"
2) "10.201.37.5:10004"
redis 127.0.0.1:6379> get /cname/ulricqin.com
"/rs/demo.xae.xiaomi.com"
```

每隔reload_uri_interval（单位s，默认5s），从redis重新加载路由表

## 配置项说明

- **redis_server**: DINP server模块的redis server地址
- **reload_uri_interval**: 更新路由表的周期，单位s (默认5s)
其它配置项与安装同CloudFoundry gorouter。
