-- =============================================
-- 诊所预约挂号系统 - 数据库初始化脚本
-- =============================================
-- 表结构说明：
--   departments      - 科室表
--   doctors          - 医生表
--   schedule_templates - 排班模板（周期性排班规则）
--   schedule_slots   - 排班号源（具体日期时段的号源）
--   appointments     - 预约单
--   appointment_logs - 预约操作日志
--   suspension_days  - 停诊记录
-- =============================================

-- 预约状态枚举
-- pending:    待就诊（已预约未取号）
-- confirmed:  已确认（已取号）
-- completed:  已完成（已就诊）
-- cancelled:  已取消（患者取消）
-- expired:    已过期（未取号自动过期）
-- suspended:  已停诊（医生停诊取消）
CREATE TYPE appointment_status AS ENUM (
    'pending',
    'confirmed',
    'completed',
    'cancelled',
    'expired',
    'suspended'
);

-- 周几枚举，用于排班模板
CREATE TYPE day_of_week AS ENUM (
    'monday',
    'tuesday',
    'wednesday',
    'thursday',
    'friday',
    'saturday',
    'sunday'
);

-- =============================================
-- 科室表
-- =============================================
CREATE TABLE departments (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE departments IS '科室表';
COMMENT ON COLUMN departments.id IS '科室ID';
COMMENT ON COLUMN departments.name IS '科室名称';
COMMENT ON COLUMN departments.description IS '科室描述';

-- =============================================
-- 医生表
-- =============================================
CREATE TABLE doctors (
    id            BIGSERIAL PRIMARY KEY,
    department_id BIGINT NOT NULL REFERENCES departments(id),
    name          VARCHAR(100) NOT NULL,
    title         VARCHAR(50),
    description   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE doctors IS '医生表';
COMMENT ON COLUMN doctors.id IS '医生ID';
COMMENT ON COLUMN doctors.department_id IS '所属科室ID';
COMMENT ON COLUMN doctors.name IS '医生姓名';
COMMENT ON COLUMN doctors.title IS '职称（主任医师、副主任医师等）';

-- =============================================
-- 排班模板表
-- 定义医生周期性的出诊规则
-- =============================================
CREATE TABLE schedule_templates (
    id          BIGSERIAL PRIMARY KEY,
    doctor_id   BIGINT NOT NULL REFERENCES doctors(id),
    day_of_week day_of_week NOT NULL,
    start_time  TIME NOT NULL,
    end_time    TIME NOT NULL,
    quota       INT NOT NULL DEFAULT 10,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(doctor_id, day_of_week, start_time, end_time)
);

COMMENT ON TABLE schedule_templates IS '排班模板表';
COMMENT ON COLUMN schedule_templates.doctor_id IS '医生ID';
COMMENT ON COLUMN schedule_templates.day_of_week IS '周几出诊';
COMMENT ON COLUMN schedule_templates.start_time IS '开始时间';
COMMENT ON COLUMN schedule_templates.end_time IS '结束时间';
COMMENT ON COLUMN schedule_templates.quota IS '该时段号源数量';

-- =============================================
-- 排班号源表
-- 具体某一天某一时段的实际号源
-- =============================================
CREATE TABLE schedule_slots (
    id           BIGSERIAL PRIMARY KEY,
    doctor_id    BIGINT NOT NULL REFERENCES doctors(id),
    schedule_date DATE NOT NULL,
    start_time   TIME NOT NULL,
    end_time     TIME NOT NULL,
    total_quota  INT NOT NULL,
    used_quota   INT NOT NULL DEFAULT 0,
    is_suspended BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(doctor_id, schedule_date, start_time, end_time)
);

COMMENT ON TABLE schedule_slots IS '排班号源表';
COMMENT ON COLUMN schedule_slots.doctor_id IS '医生ID';
COMMENT ON COLUMN schedule_slots.schedule_date IS '出诊日期';
COMMENT ON COLUMN schedule_slots.start_time IS '开始时间';
COMMENT ON COLUMN schedule_slots.end_time IS '结束时间';
COMMENT ON COLUMN schedule_slots.total_quota IS '总号源数';
COMMENT ON COLUMN schedule_slots.used_quota IS '已预约数';
COMMENT ON COLUMN schedule_slots.is_suspended IS '是否停诊';

-- 号源库存约束：已预约数不能超过总号源数
-- 这是数据库层面防止超卖的核心约束
ALTER TABLE schedule_slots ADD CONSTRAINT quota_check CHECK (used_quota >= 0 AND used_quota <= total_quota);

-- =============================================
-- 预约单表
-- =============================================
CREATE TABLE appointments (
    id              BIGSERIAL PRIMARY KEY,
    slot_id         BIGINT NOT NULL REFERENCES schedule_slots(id),
    patient_name    VARCHAR(100) NOT NULL,
    patient_phone   VARCHAR(20) NOT NULL,
    patient_id_card VARCHAR(18),
    status          appointment_status NOT NULL DEFAULT 'pending',
    appointment_no  VARCHAR(32) UNIQUE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE appointments IS '预约单表';
COMMENT ON COLUMN appointments.id IS '预约ID';
COMMENT ON COLUMN appointments.slot_id IS '号源ID';
COMMENT ON COLUMN appointments.patient_name IS '患者姓名';
COMMENT ON COLUMN appointments.patient_phone IS '患者手机号';
COMMENT ON COLUMN appointments.patient_id_card IS '患者身份证号';
COMMENT ON COLUMN appointments.status IS '预约状态';
COMMENT ON COLUMN appointments.appointment_no IS '预约单号';

CREATE INDEX idx_appointments_slot_id ON appointments(slot_id);
CREATE INDEX idx_appointments_status ON appointments(status);
CREATE INDEX idx_appointments_patient_phone ON appointments(patient_phone);
CREATE INDEX idx_appointments_created_at ON appointments(created_at);

-- =============================================
-- 预约操作日志表
-- 记录所有预约状态变更，便于追溯
-- =============================================
CREATE TABLE appointment_logs (
    id              BIGSERIAL PRIMARY KEY,
    appointment_id  BIGINT NOT NULL REFERENCES appointments(id),
    old_status      appointment_status,
    new_status      appointment_status NOT NULL,
    operator        VARCHAR(100) NOT NULL,
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE appointment_logs IS '预约操作日志表';
COMMENT ON COLUMN appointment_logs.appointment_id IS '预约ID';
COMMENT ON COLUMN appointment_logs.old_status IS '变更前状态';
COMMENT ON COLUMN appointment_logs.new_status IS '变更后状态';
COMMENT ON COLUMN appointment_logs.operator IS '操作人';
COMMENT ON COLUMN appointment_logs.reason IS '变更原因';

-- =============================================
-- 停诊记录表
-- =============================================
CREATE TABLE suspension_days (
    id         BIGSERIAL PRIMARY KEY,
    doctor_id  BIGINT NOT NULL REFERENCES doctors(id),
    suspend_date DATE NOT NULL,
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(doctor_id, suspend_date)
);

COMMENT ON TABLE suspension_days IS '停诊记录表';
COMMENT ON COLUMN suspension_days.doctor_id IS '医生ID';
COMMENT ON COLUMN suspension_days.suspend_date IS '停诊日期';
COMMENT ON COLUMN suspension_days.reason IS '停诊原因';

-- =============================================
-- 自动更新时间戳的触发器函数
-- =============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_departments_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_doctors_updated_at
    BEFORE UPDATE ON doctors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_schedule_templates_updated_at
    BEFORE UPDATE ON schedule_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_schedule_slots_updated_at
    BEFORE UPDATE ON schedule_slots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_appointments_updated_at
    BEFORE UPDATE ON appointments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================
-- 预约状态变更时自动记录日志的触发器
-- =============================================
CREATE OR REPLACE FUNCTION log_appointment_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status <> NEW.status THEN
        INSERT INTO appointment_logs (appointment_id, old_status, new_status, operator, reason)
        VALUES (NEW.id, OLD.status, NEW.status, 'system', 'status_change');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_appointment_status_log
    AFTER UPDATE ON appointments
    FOR EACH ROW
    WHEN (OLD.status <> NEW.status)
    EXECUTE FUNCTION log_appointment_status_change();

-- =============================================
-- 插入测试数据
-- =============================================

-- 科室
INSERT INTO departments (name, description) VALUES
('内科', '诊治内科常见疾病，包括呼吸、消化、心血管等系统'),
('外科', '普通外科疾病诊治，手术治疗'),
('儿科', '儿童常见病诊治，0-14岁儿童健康管理'),
('妇产科', '妇科疾病诊治、孕期保健、产科服务'),
('骨科', '骨骼关节疾病诊治，运动损伤康复');

-- 医生
INSERT INTO doctors (department_id, name, title, description) VALUES
(1, '张医生', '主任医师', '从事内科临床工作20年，擅长心血管疾病'),
(1, '李医生', '副主任医师', '呼吸系统疾病专家'),
(2, '王医生', '主任医师', '普外科微创手术专家'),
(3, '刘医生', '主治医师', '儿童常见病、多发病诊治'),
(4, '陈医生', '副主任医师', '妇产科常见病、孕期保健'),
(5, '赵医生', '主任医师', '骨科创伤、关节置换专家');

-- 排班模板
INSERT INTO schedule_templates (doctor_id, day_of_week, start_time, end_time, quota) VALUES
(1, 'monday',    '08:00', '12:00', 20),
(1, 'wednesday', '08:00', '12:00', 20),
(1, 'friday',    '14:00', '17:00', 15),
(2, 'tuesday',   '08:00', '12:00', 20),
(2, 'thursday',  '08:00', '12:00', 20),
(3, 'monday',    '08:00', '17:00', 30),
(3, 'wednesday', '08:00', '17:00', 30),
(3, 'friday',    '08:00', '17:00', 30),
(4, 'monday',    '08:00', '12:00', 25),
(4, 'tuesday',   '08:00', '12:00', 25),
(4, 'wednesday', '08:00', '12:00', 25),
(5, 'thursday',  '08:00', '17:00', 20),
(5, 'friday',    '08:00', '17:00', 20),
(6, 'tuesday',   '08:00', '17:00', 15),
(6, 'thursday',  '08:00', '17:00', 15);
