FROM nginx:alpine

# Copy frontend files
COPY lymuru-web/index.html /usr/share/nginx/html/
COPY lymuru-web/styles.css /usr/share/nginx/html/
COPY lymuru-web/app.js /usr/share/nginx/html/

# Copy image assets
COPY assets /usr/share/nginx/html/assets

# Copy nginx config
COPY lymuru-web/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
