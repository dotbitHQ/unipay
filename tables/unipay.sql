CREATE DATABASE `unipay_db`;
USE `unipay_db`;

-- t_order_info
CREATE TABLE `t_order_info`
(
    `id`           BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `order_id`     VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `business_id`  VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `address`      VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `algorithm_id` smallint(6)         NOT NULL DEFAULT '0' COMMENT '3,5-EVM 4-TRON 7-DOGE',
    `amount`       DECIMAL(60)         NOT NULL DEFAULT '0' COMMENT 'Order Amount',
    `pay_token_id` VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `pay_status`   smallint(6)         NOT NULL DEFAULT '0' COMMENT '0-Unpaid 1-Confirm',
    `order_status` smallint(6)         NOT NULL DEFAULT '0' COMMENT '0-Normal 1-Cancel',
    `created_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `uk_order_id` (`order_id`) USING BTREE,
    KEY `k_address` (`address`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='order info';

-- t_payment_info
CREATE TABLE `t_payment_info`
(
    `id`              BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `order_id`        VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `address`         VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `algorithm_id`    smallint(6)         NOT NULL DEFAULT '0' COMMENT '3,5-EVM 4-TRON 7-DOGE',
    `pay_hash`        VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `timestamp`       BIGINT              NOT NULL DEFAULT '0' COMMENT '',
    `pay_hash_status` smallint(6)         NOT NULL DEFAULT '0' COMMENT '0-Pending 1-Confirm 2-Fail',
    `refund_status`   smallint(6)         NOT NULL DEFAULT '0' COMMENT '0-Default 1-refunded',
    `refund_hash`     VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `refund_nonce`    INT                 NOT NULL DEFAULT '0' COMMENT '',
    `created_at`      TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`      TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `uk_order_id` (`order_id`) USING BTREE,
    KEY `k_address` (`address`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='order info';