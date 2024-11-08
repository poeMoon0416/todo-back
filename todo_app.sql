DROP TABLE IF EXISTS todos;

-- MySQLではBOOLはTINYINT(1バイト)として扱われるようだ
-- MySQL公式 / https://dev.mysql.com/doc/refman/8.0/ja/other-vendor-data-types.html
CREATE TABLE todos (
  id INT PRIMARY KEY AUTO_INCREMENT,
  title VARCHAR(255) NOT NULL,
  detail VARCHAR(255) NOT NULL,
  point INT NOT NULL,
  done BOOL NOT NULL
);

INSERT INTO todos(title, detail, point, done) VALUES ('test_title', 'test_detail', 0, false);
