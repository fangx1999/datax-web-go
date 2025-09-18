-- ========== DataX Web 管理平台数据库初始化脚本 ==========
CREATE DATABASE IF NOT EXISTS `datax_web` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `datax_web`;

-- 用户表 - 存储系统用户信息，支持管理员和普通用户角色
-- 默认管理员账户：
-- 用户名：admin
-- 密码：admin (MD5加密存储)
-- 首次登录后请立即修改密码
DROP TABLE IF EXISTS `users`;
CREATE TABLE IF NOT EXISTS `users`
(
    `id`         BIGINT UNSIGNED       NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '用户ID，主键',
    `username`   VARCHAR(64)           NOT NULL UNIQUE COMMENT '用户名，唯一标识',
    `password`   VARCHAR(255)          NOT NULL COMMENT '密码，BCrypt加密存储',
    `role`       ENUM ('admin','user') NOT NULL DEFAULT 'user' COMMENT '用户角色：admin管理员，user普通用户',
    `disabled`   TINYINT(1)            NOT NULL DEFAULT 0 COMMENT '是否禁用：0启用，1禁用',
    `created_by` INT                            DEFAULT NULL COMMENT '创建者用户ID',
    `updated_by` INT                            DEFAULT NULL COMMENT '更新者用户ID',
    `created_at` TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

INSERT INTO `users` (`username`, `password`, `role`)
VALUES ('admin', '$2a$10$p/Yw3z/8VnHL7t84oRhg5upY2bY6sJxhS3OMeEvkLckR2Brkuzg/y', 'admin');


-- 数据源表 - 存储各种类型的数据源连接信息
DROP TABLE IF EXISTS `data_sources`;
CREATE TABLE `data_sources`
(
    `id`           INT AUTO_INCREMENT PRIMARY KEY COMMENT '数据源ID，主键',
    `name`         VARCHAR(100)                       NOT NULL COMMENT '数据源名称',
    `type`         ENUM ('mysql','ofs','hdfs','cosn') NOT NULL COMMENT '数据源类型：mysql数据库，ofs对象存储，hdfs分布式文件系统，cosn腾讯云对象存储',
    -- MySQL fields
    `db_url`       VARCHAR(255) DEFAULT NULL COMMENT '数据库连接URL，仅MySQL类型使用',
    `db_user`      VARCHAR(50)  DEFAULT NULL COMMENT '数据库用户名，仅MySQL类型使用',
    `db_password`  VARCHAR(100) DEFAULT NULL COMMENT '数据库密码，仅MySQL类型使用',
    `db_database`  VARCHAR(100) DEFAULT NULL COMMENT '数据库名称，仅MySQL类型使用',
    -- Unified Hadoop-compatible storage config
    `defaultfs`    VARCHAR(255) DEFAULT NULL COMMENT 'Hadoop默认文件系统地址，用于HDFS/OFS/COSN类型',
    `hadoopconfig` TEXT         DEFAULT NULL COMMENT 'Hadoop配置信息JSON，用于HDFS/OFS/COSN类型',
    `created_by`   INT          DEFAULT NULL COMMENT '创建者用户ID',
    `updated_by`   INT          DEFAULT NULL COMMENT '更新者用户ID',
    `created_at`   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;


-- 任务表 - 存储DataX任务配置信息
DROP TABLE IF EXISTS `tasks`;
CREATE TABLE `tasks`
(
    `id`          INT AUTO_INCREMENT PRIMARY KEY COMMENT '任务ID，主键',
    `name`        VARCHAR(100) NOT NULL COMMENT '任务名称',
    `source_id`   INT          NOT NULL COMMENT '源数据源ID，关联data_sources表',
    `target_id`   INT          NOT NULL COMMENT '目标数据源ID，关联data_sources表',
    `json_config` MEDIUMTEXT COMMENT 'DataX任务配置JSON，包含reader和writer配置',
    `created_by`  INT      DEFAULT NULL COMMENT '创建者用户ID',
    `updated_by`  INT      DEFAULT NULL COMMENT '更新者用户ID',
    `created_at`  TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;


-- 任务日志表 - 统一的执行日志存储
-- 支持独立任务执行和任务流步骤执行两种上下文
DROP TABLE IF EXISTS `task_logs`;
CREATE TABLE `task_logs`
(
    `id`                INT AUTO_INCREMENT PRIMARY KEY COMMENT '日志ID，主键',
    `task_id`           INT                                                              NOT NULL COMMENT '任务ID，关联tasks表',
    `flow_execution_id` INT                                                                       DEFAULT NULL COMMENT '任务流执行ID，如果为NULL则表示独立任务执行',
    `step_id`           INT                                                                       DEFAULT NULL COMMENT '任务流步骤ID，如果为NULL则表示独立任务执行',
    `step_order`        INT                                                                       DEFAULT NULL COMMENT '步骤顺序，用于任务流中的步骤排序',
    `execution_type`    ENUM ('scheduled','manual')                                      NOT NULL DEFAULT 'manual' COMMENT '执行类型：scheduled定时执行，manual手动执行',
    `start_time`        TIMESTAMP                                    NOT NULL COMMENT '开始执行时间',
    `end_time`          TIMESTAMP                                           COMMENT '结束执行时间，NULL表示仍在运行',
    `status`            ENUM ('pending','running','success','failed','killed','skipped') NOT NULL DEFAULT 'pending' COMMENT '执行状态：pending等待，running运行中，success成功，failed失败，killed已终止，skipped跳过',
    `log`               MEDIUMTEXT                                   NOT NULL  COMMENT '执行日志内容',
    `created_at`        TIMESTAMP                                                                 DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_flow_execution` (`flow_execution_id`),
    INDEX `idx_step_id` (`step_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_start_time` (`start_time`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;


-- 任务流表 - 管理定时调度的任务流
DROP TABLE IF EXISTS `task_flows`;
CREATE TABLE `task_flows`
(
    `id`          INT AUTO_INCREMENT PRIMARY KEY COMMENT '任务流ID，主键',
    `name`        VARCHAR(100) NOT NULL COMMENT '任务流名称',
    `description` TEXT COMMENT '任务流描述',
    `cron_expr`   VARCHAR(100) NOT NULL COMMENT 'Cron表达式，定义定时执行规则',
    `enabled`     TINYINT(1)   NOT NULL DEFAULT 1 COMMENT '是否启用：1启用，0禁用',
    `created_by`  INT                   DEFAULT NULL COMMENT '创建者用户ID',
    `updated_by`  INT                   DEFAULT NULL COMMENT '更新者用户ID',
    `created_at`  TIMESTAMP             DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP             DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

-- 任务流步骤表 - 定义任务流中任务的执行顺序
DROP TABLE IF EXISTS `task_flow_steps`;
CREATE TABLE `task_flow_steps`
(
    `id`              INT AUTO_INCREMENT PRIMARY KEY COMMENT '步骤ID，主键',
    `flow_id`         INT NOT NULL COMMENT '任务流ID，关联task_flows表',
    `task_id`         INT NOT NULL COMMENT '任务ID，关联tasks表',
    `step_order`      INT NOT NULL COMMENT '步骤顺序，从1开始递增',
    `timeout_minutes` INT      DEFAULT NULL COMMENT '超时时间（分钟），NULL表示不限制',
    `created_by`      INT      DEFAULT NULL COMMENT '创建者用户ID',
    `updated_by`      INT      DEFAULT NULL COMMENT '更新者用户ID',
    `created_at`      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE KEY `uk_flow_step` (`flow_id`, `step_order`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

-- 任务流执行记录表 - 存储任务流的执行历史
DROP TABLE IF EXISTS `task_flow_executions`;
CREATE TABLE `task_flow_executions`
(
    `id`             INT AUTO_INCREMENT PRIMARY KEY COMMENT '执行记录ID，主键',
    `flow_id`        INT                                          NOT NULL COMMENT '任务流ID，关联task_flows表',
    `status`         ENUM ('running','success','failed','killed') NOT NULL COMMENT '执行状态：running运行中，success成功，failed失败，killed已终止',
    `execution_type` ENUM ('scheduled','manual')                  NOT NULL DEFAULT 'scheduled' COMMENT '执行类型：scheduled定时执行，manual手动执行',
    `start_time`     TIMESTAMP                                    NOT NULL COMMENT '开始执行时间',
    `end_time`       TIMESTAMP                                             COMMENT '结束执行时间，NULL表示仍在运行',
    `created_at`     TIMESTAMP                                             DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;