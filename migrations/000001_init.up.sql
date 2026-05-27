CREATE TABLE IF NOT EXISTS subscriptions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(255) NOT NULL,
    price        INTEGER      NOT NULL,
    user_id      UUID         NOT NULL,
    start_date   DATE         NOT NULL,
    end_date     DATE,

    CONSTRAINT ck_price_positive     CHECK (price > 0),
    CONSTRAINT ck_end_after_start    CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS ix_subscriptions_user_id      ON subscriptions (user_id);
CREATE INDEX IF NOT EXISTS ix_subscriptions_service_name ON subscriptions (service_name);
CREATE INDEX IF NOT EXISTS ix_subscriptions_user_service ON subscriptions (user_id, service_name);
