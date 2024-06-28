* [API List](#api-list)
    * [Get Version](#Get-Version)
    * [Get Order Info](#Get-Order-Info)
    * [Get Payment Info](#Get-Payment-Info)
    * [Order Create](#Order-Create)
    * [Order Refund](#Order-Refund)

* [Error](#error)
    * [Error Example](#error-example)
    * [Error Code](#error-code)


## API List

Please familiarize yourself with the meaning of some common parameters before reading the API list:

| param                                                                                   | description                                        |
|:----------------------------------------------------------------------------------------|:---------------------------------------------------|
| type                                                                                    | Filled with "blockchain" for now                   |
| coin_type <sup>[1](https://github.com/satoshilabs/slips/blob/master/slip-0044.md)</sup> | 60: eth, 195: trx, 9006: bsc, 966: matic, 3: doge  |
| account                                                                                 | Contains the suffix `.bit` in it                   |
| key                                                                                     | Generally refers to the blockchain address for now |



### Get Version

**Request**
* path: `/v1/version`
* param: none

**Response**

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "version": ""
  }
}
```

**Usage**

```shell
curl -X POST localhost/v1/server/info
```


### Get Order Info

**Request**
* path: `/v1/order/info`
* param:
```json
{
  "business_id": "",
  "order_id": ""
}
```
**Response**

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "order_id": "",
    "payment_address": "",
    "contract_address": "",
    "client_secret": ""
  }
}
```

**Usage**

```shell
curl -X POST localhsot/v1/order/info -d'{"business_id": "","order_id": ""}'
```

### Get Payment Info

**Request**
* path: `/v1/order/info`
* param:
```json
{
  "business_id": "",
  "order_id_list": [],
  "pay_hash_list": []
}
```
**Response**

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "payment_list": [
      {
        "order_id": "",
        "pay_hash": "",
        "source_payment": "",
        "pay_address": "",
        "amount": 0.00,
        "algorithm_id": 0,
        "pay_hash_status": 0,
        "refund_hash": "",
        "refund_status": 0,
        "payment_address": "",
        "contract_address": ""
      }
    ]
  }
}
```

**Usage**

```shell
curl -X POST localhsot/v1/payment/info -d'{"business_id": "","order_id_list": [],"pay_hash_list": []}'
```


### Order Create

**Request**
* path: `/v1/order/crate`
* param:
```json
{
  "type": "blockchain",
  "key_info": {
    "coin_type": "60",
    "key": "0x111..."
  },
  "business_id": "",
  "amount": 0.00,
  "pay_token_id": "eth_eth",
  "payment_address": "",
  "premium_percentage": 0.00,
  "premium_base": 0.00,
  "premium_amount": 0.00,
  "meta_data": {
  }
}
```
**Response**

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": {
    "payment_list": [
      {
        "order_id": "",
        "payment_address": "",
        "contract_address": "",
        "stripe_payment_intent_id": "",
        "client_secret": ""
      }
    ]
  }
}
```

**Usage**

```shell
curl -X POST localhsot/v1/order/create -d'{"type":"blockchain","key_info":{"coin_type":"60","key":"0x111..."},"business_id":"","amount":0.00,"pay_token_id":"eth_eth","payment_address":"","premium_percentage":0.00,"premium_base":0.00,"premium_amount":0.00,"meta_data":{}}'
```

### Order Refund

**Request**
* path: `/v1/order/refund`
* param:
```json
{
  "business_id": "",
  "amount": 0.00,
  "refund_list": [
    {
      "order_id": "",
      "pay_hash": ""
    }
  ]
}
```
**Response**

```json
{
  "err_no": 0,
  "err_msg": "",
  "data": null
}
```

**Usage**

```shell
curl -X POST localhsot/v1/order/refund -d'{"business_id":"","amount":0.00,"refund_list":[{"order_id":"","pay_hash":""}]}'
```


## Error
### Error Example
```json
{
  "err_no": 20007,
  "err_msg": "account not exist",
  "data": null
}
```
### Error Code
```go

const (
  ApiCodeSuccess              Code = 0
  ApiCodeError500             Code = 500
  ApiCodeParamsInvalid        Code = 10000
  ApiCodeMethodNotExist       Code = 10001
  ApiCodeDbError              Code = 10002
  
  ApiCodeAccountFormatInvalid Code = 20006
  ApiCodeAccountNotExist      Code = 20007
)

```
    
