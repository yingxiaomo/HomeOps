# 🌐 自建 Telegram 接口反代

为了在国内服务器上顺畅运行机器人，推荐使用 Cloudflare 云函数搭建免费的反代服务。

## 🛠 搭建步骤

1. 登录 [Cloudflare 控制台](https://dash.cloudflare.com/)。
2. 进入 **Workers 和 Pages** -> **创建应用程序** -> **从Hello World开始**。
3. 为您的程序起个名字（如 `tg-proxy`），点击 **部署**。
4. 点击 **编辑代码**，将以下内容完全覆盖掉原有代码：

```javascript
const tg_host = "api.telegram.org";

addEventListener('fetch', event => {
    event.respondWith(handleRequest(event.request))
})

async function handleRequest(request) {
    var u = new URL(request.url);
    u.host = tg_host;
    var req = new Request(u, {
        method: request.method,
        headers: request.headers,
        body: request.body
    });
    
    const result = await fetch(req);
    return result;
}
```

5. 点击 **保存并部署**。
6. 在 **设置** 中，您可以看到分配的域名，也可以设置自定义域名。

## ⚙️ 如何在项目中使用

在您的环境配置文件中，将域名填入以下位置：

```env
# 注意：必须以 /bot 结尾
TG_BASE_URL=https://您的域名/bot
```
