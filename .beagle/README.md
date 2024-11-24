# neko

<https://github.com/m1k1o/neko>

```bash
git remote add upstream git@github.com:m1k1o/neko.git

git fetch upstream

git merge upstream/v3
```

## images

```bash
docker pull m1k1o/neko:google-chrome && \
docker tag m1k1o/neko:google-chrome registry.cn-qingdao.aliyuncs.com/wod/neko:google-chrome && \
docker push registry.cn-qingdao.aliyuncs.com/wod/neko:google-chrome

docker pull ghcr.io/m1k1o/neko/nvidia-google-chrome:2.8.11 && \
docker tag ghcr.io/m1k1o/neko/nvidia-google-chrome:2.8.11 registry.cn-qingdao.aliyuncs.com/wod/neko:nvidia-google-chrome-2.8.11 && \
docker push registry.cn-qingdao.aliyuncs.com/wod/neko:nvidia-google-chrome-2.8.11

docker pull ghcr.io/m1k1o/neko/vlc:2.8.11 && \
docker tag ghcr.io/m1k1o/neko/vlc:2.8.11 registry.cn-qingdao.aliyuncs.com/wod/neko:vlc-2.8.11 && \
docker push registry.cn-qingdao.aliyuncs.com/wod/neko:vlc-2.8.11

docker pull ghcr.io/m1k1o/neko/xfce:2.8.11 && \
docker tag ghcr.io/m1k1o/neko/xfce:2.8.11 registry.cn-qingdao.aliyuncs.com/wod/neko:xfce-2.8.11 && \
docker push registry.cn-qingdao.aliyuncs.com/wod/neko:xfce-2.8.11

docker pull ghcr.io/m1k1o/neko/kde:2.8.11 && \
docker tag ghcr.io/m1k1o/neko/kde:2.8.11 registry.cn-qingdao.aliyuncs.com/wod/neko:kde-2.8.11 && \
docker push registry.cn-qingdao.aliyuncs.com/wod/neko:kde-2.8.11
```

## deploy

```bash
sudo apt update
sudo apt install coturn -y

turnutils_uclient -u neko -w neko -y 47.104.105.222

docker rm -f neko
docker run -d \
  --name neko \
  --cap-add=SYS_ADMIN \
  --restart unless-stopped \
  --shm-size 2gb \
  -p 8080:8080 \
  -p 52060-52100:52060-52100/udp \
  -e NEKO_SCREEN=1920x1080@30 \
  -e NEKO_PASSWORD=neko \
  -e NEKO_PASSWORD_ADMIN=admin \
  -e NEKO_EPR=52060-52100 \
  -e NEKO_ICELITE=1 \
  -e NEKO_NAT1TO1=192.168.1.201 \
  registry.cn-qingdao.aliyuncs.com/wod/neko:google-chrome
```
