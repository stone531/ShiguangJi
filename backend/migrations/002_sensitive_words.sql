-- 002: 敏感词初始词表（spec §4.8 / §5.5；后台站点设置可扩展，换行分隔）
-- 词表刻意保持最小集：个人博客先审后显为主，词表仅拦截明显辱骂。
INSERT IGNORE INTO `site_settings` (`setting_key`, `setting_value`) VALUES
  ('sensitive_words', '傻逼\n煞笔\n妈的\n他妈的\n滚蛋\n废物\n垃圾人\n白痴\n脑残\n弱智');
