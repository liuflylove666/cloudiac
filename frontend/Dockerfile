FROM node:22 AS builder
WORKDIR /workspace
RUN npm i -g npm@11 \
    && npm config set registry https://registry.npmjs.org/ \
    && npm config set replace-registry-host always

COPY . .
# 构建前端执行代码包
RUN npm ci --legacy-peer-deps \
    && npm run build:vendor \
    && npm run build

# 打包镜像
FROM nginx:stable-alpine
RUN rm /etc/nginx/conf.d/default.conf
WORKDIR /usr/nginx/cloudiac-web
COPY --from=builder /workspace/build /usr/nginx/cloudiac-web
COPY --from=builder /workspace/iac.conf /etc/nginx/conf.d/
RUN chown -R nginx:nginx /usr/nginx/cloudiac-web
