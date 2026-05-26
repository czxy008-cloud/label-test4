-- =============================================
-- 候补排队功能 - 数据库迁移脚本
-- =============================================

-- 候补状态枚举
-- waiting:    等待中
-- promoted:   已递补（已自动转为预约）
-- cancelled:  已取消（患者主动取消候补）
-- expired:    已过期（就诊时间已过）
CREATE TYPE waitlist_status AS ENUM (
    'waiting',
    'promoted',
    'cancelled',
    'expired'
);

-- =============================================
-- 候补队列表
-- 记录患者的候补排队信息
-- =============================================
CREATE TABLE waitlists (
    id              BIGSERIAL PRIMARY KEY,
    slot_id         BIGINT NOT NULL REFERENCES schedule_slots(id),
    patient_name    VARCHAR(100) NOT NULL,
    patient_phone   VARCHAR(20) NOT NULL,
    patient_id_card VARCHAR(18),
    status          waitlist_status NOT NULL DEFAULT 'waiting',
    position        INT NOT NULL,
    appointment_id  BIGINT REFERENCES appointments(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE waitlists IS '候补队列表';
COMMENT ON COLUMN waitlists.id IS '候补ID';
COMMENT ON COLUMN waitlists.slot_id IS '号源ID';
COMMENT ON COLUMN waitlists.patient_name IS '患者姓名';
COMMENT ON COLUMN waitlists.patient_phone IS '患者手机号';
COMMENT ON COLUMN waitlists.patient_id_card IS '患者身份证号';
COMMENT ON COLUMN waitlists.status IS '候补状态';
COMMENT ON COLUMN waitlists.position IS '候补位置序号（用于排序）';
COMMENT ON COLUMN waitlists.appointment_id IS '递补成功后的预约单ID';
COMMENT ON COLUMN waitlists.created_at IS '创建时间';
COMMENT ON COLUMN waitlists.updated_at IS '更新时间';

-- 索引优化查询
CREATE INDEX idx_waitlists_slot_id ON waitlists(slot_id);
CREATE INDEX idx_waitlists_status ON waitlists(status);
CREATE INDEX idx_waitlists_patient_phone ON waitlists(patient_phone);
CREATE INDEX idx_waitlists_slot_status ON waitlists(slot_id, status);
CREATE UNIQUE INDEX idx_waitlists_slot_patient_active ON waitlists(slot_id, patient_phone)
    WHERE status = 'waiting';

-- =============================================
-- 自动更新时间戳的触发器
-- =============================================
CREATE TRIGGER update_waitlists_updated_at
    BEFORE UPDATE ON waitlists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================
-- 通知事件类型枚举
-- =============================================
CREATE TYPE notification_type AS ENUM (
    'waitlist_promoted',
    'appointment_cancelled',
    'appointment_confirmed',
    'appointment_suspended'
);

-- =============================================
-- 队列表（用于消息通知）
-- =============================================
CREATE TABLE notification_queue (
    id              BIGSERIAL PRIMARY KEY,
    type            notification_type NOT NULL,
    recipient_phone VARCHAR(20) NOT NULL,
    recipient_name  VARCHAR(100) NOT NULL,
    content         TEXT NOT NULL,
    metadata        JSONB,
    is_processed    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at    TIMESTAMPTZ
);

COMMENT ON TABLE notification_queue IS '通知队列表';
COMMENT ON COLUMN notification_queue.id IS '通知ID';
COMMENT ON COLUMN notification_queue.type IS '通知类型';
COMMENT ON COLUMN notification_queue.recipient_phone IS '接收人手机号';
COMMENT ON COLUMN notification_queue.recipient_name IS '接收人姓名';
COMMENT ON COLUMN notification_queue.content IS '通知内容';
COMMENT ON COLUMN notification_queue.metadata IS '附加元数据';
COMMENT ON COLUMN notification_queue.is_processed 是否已处理;
COMMENT ON COLUMN notification_queue.created_at IS '创建时间';
COMMENT ON COLUMN notification_queue.processed_at IS '处理时间';

CREATE INDEX idx_notification_queue_processed ON notification_queue(is_processed);
CREATE INDEX idx_notification_queue_created ON notification_queue(created_at);
