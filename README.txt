wget https://go.dev/dl/go1.19.13.linux-amd64.tar.gz
sudo -i
cd /home/david
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.13.linux-amd64.tar.gz

go env GOROOT
GoLand: \\wsl$\Debian\usr\local\go

sudo chmod -R 755 /usr/local/go