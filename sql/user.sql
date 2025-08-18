CREATE TABLE "user" (
                        id SERIAL PRIMARY KEY,
                        nickname VARCHAR(50),
                        password VARCHAR(255) NOT NULL,
                        email VARCHAR(100) NOT NULL,
                        avatar_url VARCHAR(255) DEFAULT NULL,
                        mobile VARCHAR(15) DEFAULT NULL,
                        is_deleted BOOLEAN DEFAULT FALSE,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        created_by INT DEFAULT NULL,
                        updated_by INT DEFAULT NULL,
                        UNIQUE (id, email)
);
