### Only works on linux

sudo apt update
sudo apt install -y \
 build-essential \
 ninja-build \
 python3-pip \
 pkg-config \
 libglib2.0-dev \
 libexpat1-dev \
 librsvg2-dev \
 libpng-dev \
 libjpeg-dev \
 libtiff-dev \
 libexif-dev \
 liblcms2-dev \
 libheif-dev \
 libwebp-dev

sudo apt update
sudo apt install -y meson ninja-build

cd /tmp
wget https://github.com/libvips/libvips/releases/download/v8.18.0/vips-8.18.0.tar.xz
tar xf vips-8.18.0.tar.xz
cd vips-8.18.0

cd /tmp/vips-8.15.1

meson setup build --libdir=lib --buildtype=release

# Compile

cd build
meson compile

# Install to the system

sudo meson install

# Refresh library cache

sudo ldconfig

# Refresh Shell Hash

hash -r
