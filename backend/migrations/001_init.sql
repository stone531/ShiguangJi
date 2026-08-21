-- 001_init.sql — 初始建表（对应 spec §4 表字典）
-- 要求 MySQL 8.0+（utf8mb4_0900_ai_ci）
-- 引擎 InnoDB，字符集 utf8mb4，排序规则 utf8mb4_0900_ai_ci

-- 4.1 admins 管理员
CREATE TABLE IF NOT EXISTS `admins` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(50) NOT NULL,
  `password_hash` VARCHAR(100) NOT NULL,
  `nickname` VARCHAR(50) NOT NULL DEFAULT '',
  `avatar` VARCHAR(255) DEFAULT NULL,
  `must_change_password` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4.3 categories 分类（先于 posts，供 FK 引用）
CREATE TABLE IF NOT EXISTS `categories` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(50) NOT NULL,
  `sort` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4.2 posts 内容表
CREATE TABLE IF NOT EXISTS `posts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(200) NOT NULL,
  `slug` VARCHAR(200) NOT NULL,
  `summary` VARCHAR(500) DEFAULT NULL,
  `cover` VARCHAR(255) DEFAULT NULL,
  `content` LONGTEXT NOT NULL,
  `content_html` LONGTEXT,
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 1,
  `video_url` VARCHAR(500) DEFAULT NULL,
  `category_id` BIGINT UNSIGNED DEFAULT NULL,
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0,
  `is_top` TINYINT(1) NOT NULL DEFAULT 0,
  `is_featured` TINYINT(1) NOT NULL DEFAULT 0,
  `views` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `like_count` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `comment_count` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `published_at` DATETIME DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_slug` (`slug`),
  KEY `idx_status_published` (`status`, `published_at`),
  KEY `idx_category_status` (`category_id`, `status`),
  KEY `idx_is_featured` (`is_featured`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_posts_category` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4.4 tags 标签
CREATE TABLE IF NOT EXISTS `tags` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(50) NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4.5 post_tags 文章标签关联
CREATE TABLE IF NOT EXISTS `post_tags` (
  `post_id` BIGINT UNSIGNED NOT NULL,
  `tag_id` BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (`post_id`, `tag_id`),
  KEY `idx_tag` (`tag_id`),
  CONSTRAINT `fk_post_tags_post` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_post_tags_tag` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4.6 comments 评论
CREATE TABLE IF NOT EXISTS `comments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `post_id` BIGINT UNSIGNED NOT NULL,
  `parent_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `nickname` VARCHAR(50) NOT NULL,
  `email` VARCHAR(100) DEFAULT NULL,
  `content` TEXT NOT NULL,
  `is_admin` TINYINT(1) NOT NULL DEFAULT 0,
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0,
  `ip` VARCHAR(45) NOT NULL,
  `user_agent` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_post_status_created` (`post_id`, `status`, `created_at`),
  KEY `idx_status_created` (`status`, `created_at`),
  KEY `idx_ip_created` (`ip`, `created_at`),
  CONSTRAINT `fk_comments_post` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4.7 likes 点赞
CREATE TABLE IF NOT EXISTS `likes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `post_id` BIGINT UNSIGNED NOT NULL,
  `fingerprint` CHAR(64) NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_post_fp` (`post_id`, `fingerprint`),
  CONSTRAINT `fk_likes_post` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4.8 site_settings 站点设置
CREATE TABLE IF NOT EXISTS `site_settings` (
  `setting_key` VARCHAR(50) NOT NULL,
  `setting_value` TEXT,
  PRIMARY KEY (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 预置配置键默认值（spec §4.8）
INSERT IGNORE INTO `site_settings` (`setting_key`, `setting_value`) VALUES
  ('site_name', '时光集'),
  ('site_desc', '记录技术与生活'),
  ('avatar', ''),
  ('wechat_qr', ''),
  ('alipay_qr', ''),
  ('donate_text', '写得不错，请我喝杯咖啡 ☕'),
  ('social_links', '[]'),
  ('page_size', '10'),
  ('comment_enabled', 'true');
