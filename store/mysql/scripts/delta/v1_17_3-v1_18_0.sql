/*
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */
--
-- Database: `polaris_server`
--
USE `polaris_server`;

-- 服务可见性
ALTER TABLE service
ADD COLUMN `export_to` TEXT COMMENT 'service export to some namespace';

ALTER TABLE namespace
ADD COLUMN `service_export_to` TEXT COMMENT 'namespace metadata';

ALTER TABLE namespace
ADD COLUMN `metadata` TEXT COMMENT 'namespace metadata';

ALTER TABLE config_file_release
ADD COLUMN `release_type` VARCHAR(25) NOT NULL DEFAULT '' COMMENT '文件类型：""：全量 gray：灰度';

/* 服务契约表 */
CREATE TABLE service_contract (
        `id` VARCHAR(128) NOT NULL COMMENT '服务契约主键',
        `name` VARCHAR(128) NOT NULL COMMENT '服务契约名称',
        `namespace` VARCHAR(64) NOT NULL COMMENT '命名空间',
        `service` VARCHAR(128) NOT NULL COMMENT '服务名称',
        `protocol` VARCHAR(32) NOT NULL COMMENT '当前契约对应的协议信息 e.g. http/dubbo/grpc/thrift',
        `version` VARCHAR(64) NOT NULL COMMENT '服务契约版本',
        `revision` VARCHAR(128) NOT NULL COMMENT '当前服务契约的全部内容版本摘要',
        `flag` TINYINT(4) DEFAULT 0 COMMENT '逻辑删除标志位 ， 0 位有效 ， 1 为逻辑删除',
        `content` LONGTEXT COMMENT '描述信息',
        `ctime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        `mtime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        -- 通过 服务 + 协议信息 + 契约版本 + 名称 进行一次 hash 计算，作为主键
        PRIMARY KEY (`id`),
        -- 服务 + 协议信息 + 契约版本 + 辅助标签 必须保证唯一
        KEY (
            `namespace`,
            `service`,
            `name`,
            `version`,
            `protocol`
        )
    ) ENGINE = InnoDB;

/* 服务契约中针对单个接口定义的详细信息描述表 */
CREATE TABLE service_contract_detail (
        `id` VARCHAR(128) NOT NULL COMMENT '服务契约单个接口定义记录主键',
        `contract_id` VARCHAR(128) NOT NULL COMMENT '服务契约 ID',
        `name` VARCHAR(128) NOT NULL COMMENT '接口名称',
        `method` VARCHAR(32) NOT NULL COMMENT 'http协议中的 method 字段, eg:POST/GET/PUT/DELETE, 其他 gRPC 可以用来标识 stream 类型',
        `path` VARCHAR(128) NOT NULL COMMENT '接口具体全路径描述',
        `source` INT COMMENT '该条记录来源, 0:SDK/1:MANUAL',
        `content` LONGTEXT COMMENT '描述信息',
        `revision` VARCHAR(128) NOT NULL COMMENT '当前接口定义的全部内容版本摘要',
        `flag` TINYINT(4) DEFAULT 0 COMMENT '逻辑删除标志位, 0 位有效, 1 为逻辑删除',
        `ctime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        `mtime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        PRIMARY KEY (`id`),
        -- 服务契约id + method + path + source 需保证唯一
        KEY (`contract_id`, `path`, `method`)
    ) ENGINE = InnoDB;

/* 灰度资源 */
CREATE TABLE `gray_resource`
(
    `name`        VARCHAR(128)    NOT NULL COMMENT '灰度资源',
    `match_rule`  TEXT            NOT NULL COMMENT '配置规则',
    `create_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`   VARCHAR(32)     DEFAULT "" COMMENT '创建人',
    `modify_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`   VARCHAR(32)     DEFAULT "" COMMENT '最后更新人',
    `flag`        TINYINT(4)            DEFAULT 0 COMMENT '逻辑删除标志位, 0 位有效, 1 为逻辑删除',
    PRIMARY KEY (`name`)
) ENGINE = InnoDB COMMENT = '灰度资源表';
