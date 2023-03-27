CREATE DATABASE `unipay_db`;
USE `unipay_db`;

-- t_order_info
CREATE TABLE `t_order_info`
(
    `id`           BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `order_id`     VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `business_id`  VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `pay_address`  VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `algorithm_id` SMALLINT            NOT NULL DEFAULT '0' COMMENT '3,5-EVM 4-TRON 7-DOGE',
    `amount`       DECIMAL(60)         NOT NULL DEFAULT '0' COMMENT 'Order Amount',
    `pay_token_id` VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `pay_status`   SMALLINT            NOT NULL DEFAULT '0' COMMENT '0-Unpaid 1-Paid',
    `order_status` SMALLINT            NOT NULL DEFAULT '0' COMMENT '0-Normal 1-Cancel',
    `created_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `uk_order_id` (`order_id`) USING BTREE,
    KEY `k_pay_address` (`pay_address`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='order info';

-- t_payment_info
CREATE TABLE `t_payment_info`
(
    `id`              BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `pay_hash`        VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `order_id`        VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `pay_address`     VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `algorithm_id`    SMALLINT            NOT NULL DEFAULT '0' COMMENT '3,5-EVM 4-TRON 7-DOGE',
    `timestamp`       BIGINT              NOT NULL DEFAULT '0' COMMENT '',
    `pay_hash_status` SMALLINT            NOT NULL DEFAULT '0' COMMENT '0-Pending 1-Confirm 2-Fail',
    `refund_status`   SMALLINT            NOT NULL DEFAULT '0' COMMENT '0-Default 1-UnRefunded 2-Refunded',
    `refund_hash`     VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `refund_nonce`    INT                 NOT NULL DEFAULT '0' COMMENT '',
    `created_at`      TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`      TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `uk_pay_hash` (`pay_hash`) USING BTREE,
    KEY `k_order_id` (`order_id`) USING BTREE,
    KEY `k_pay_address` (`pay_address`) USING BTREE,
    KEY `k_timestamp` (`timestamp`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='payment info';

-- t_notice_info
CREATE TABLE `t_notice_info`
(
    `id`            BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `order_id`      VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `event_type`    VARCHAR(255)        NOT NULL DEFAULT '' COMMENT 'ORDER.PAY, ORDER.REFUND',
    `notice_count`  SMALLINT            NOT NULL DEFAULT '0' COMMENT '',
    `notice_status` SMALLINT            NOT NULL DEFAULT '0' COMMENT '0-Default 1-OK',
    `timestamp`     BIGINT              NOT NULL DEFAULT '0' COMMENT '',
    `created_at`    TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`    TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `uk_order_id` (`order_id`) USING BTREE,
    KEY `k_timestamp` (`timestamp`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='notice info';

-- t_block_parse_info
CREATE TABLE `t_block_parse_info`
(
    `id`           BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `parser_type`  SMALLINT            NOT NULL DEFAULT '0' COMMENT '',
    `block_number` BIGINT(20) UNSIGNED NOT NULL DEFAULT '0' COMMENT '',
    `block_hash`   VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `parent_hash`  VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `created_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `uk_parser_number` (parser_type, block_number) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='block parse info';