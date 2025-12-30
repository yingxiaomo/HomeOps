# 使用轻量级的 Python 3.12 镜像
FROM python:3.12-slim

# 设置工作目录
WORKDIR /app

# 设置时区为上海 (可选，方便查看日志时间)
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 复制依赖文件并安装
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制项目所有文件
COPY . .

# 启动命令
CMD ["python", "main.py"]
