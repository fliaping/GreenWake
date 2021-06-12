# 功能

## 唤醒

- 打开页面保持目标主机唤醒（间歇性发送WOL包）
- 链接：http://192.168.1.2:8055/
- 通过反向代理，可以开放公网

默认http端口8055，通过设置环境变量 HTTP_PORT 可修改端口

环境变量：

- HTTP_PORT， 默认http端口8055，通过设置环境变量 HTTP_PORT 可修改端口
- HTTP_USER，http鉴权用户名
- HTTP_PASSWD，http鉴权密码
- HOST_IP，需要唤醒的主机IP
- TEL_PORT，需要唤醒的主机上的开放的tcp端口，因为睡眠时ip可能是可以ping通的，需要用telnet连接tcp端口判断主机是否被唤醒
- HOST_MAC，需要唤醒的主机mac地址

```bash
docker run -d --network=host --name=my-wol --restart=always -e HTTP_PORT=8055 -e HOST_IP=192.168.1.2 -e TEL_PORT=3389 -e HOST_MAC=xx:xx:xx:xx:xx:xx -e HTTP_USER=xx -e HTTP_PASSWD=xxx xuping/my-wol:1.1
```

## clash规则过滤

- 过滤`proxy-groups` 下的服务器，主要因为有的是n倍计费，要过滤掉
- 链接：http://192.168.1.2:8055/clashFilter?url=可以获取clash配置的地址&pattern=过滤的正则表达式
- 注意：参数需要进行 URL Encode
- 正则举例：.*(流量|备用|临时|耗尽|测试|\[2\]|\[5\]|\[1.5\]).*
- 正则URL Encode：.*(%E6%B5%81%E9%87%8F%7C%E5%A4%87%E7%94%A8%7C%E4%B8%B4%E6%97%B6%7C%E8%80%97%E5%B0%BD%7C%E6%B5%8B%E8%AF%95%7C%5C%5B2%5C%5D%7C%5C%5B5%5C%5D%7C%5C%5B1.5%5C%5D).*