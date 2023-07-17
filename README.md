* [Prerequisites](#prerequisites)
* [Install &amp; Run](#install--run)
    * [Source Compile](#source-compile)
    * [Docker](#docker)
* [Usage](#usage)

# unipay

Order service that supports payment using CKB, ETH, BNB, Matic, TRX.

## Prerequisites

* Ubuntu 18.04 or newer
* MYSQL >= 8.0
* go version >= 1.17.0
* [CKB Node](https://github.com/nervosnetwork/ckb)
* [ETH Node](https://ethereum.org/en/community/support/#building-support)
* [BSC Node](https://docs.binance.org/smart-chain/developer/fullnode.html)
* [Tron Node](https://developers.tron.network/docs/fullnode)
* [Polygon Node](https://wiki.polygon.technology/docs/pos/getting-started)

## Install & Run

### Source Compile

```bash
# get the code
git clone https://github.com/dotbitHQ/unipay.git

# edit config/config.yaml and run unipay_svr

# compile and run
cd unipay
make unipay_svr
./unipay_svr --config=config/config.yaml
```

### Docker
* docker >= 20.10
* docker-compose >= 2.2.2

```bash
sudo curl -L "https://github.com/docker/compose/releases/download/v2.2.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose
docker-compose up -d
```

_if you already have a mysql installed, just run_
```bash
docker run -dv $PWD/config/config.yaml:/app/config/config.yaml --name unipay_svr dotbitteam/unipay:latest
```