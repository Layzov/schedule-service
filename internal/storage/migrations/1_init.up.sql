-- Active: 1765628276124@@127.0.0.1@5439@rasp_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Availability Templates
CREATE TABLE IF NOT EXISTS availability_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    teacher_id TEXT NOT NULL,
    recurrence_days TEXT[] NOT NULL,
    recurrence_start_time TIME NOT NULL,
    recurrence_end_time TIME NOT NULL,
    slot_duration_minutes INTEGER NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_availability_templates_teacher_id ON availability_templates (teacher_id);
CREATE INDEX IF NOT EXISTS idx_availability_templates_enabled ON availability_templates (enabled);

-- Time Blocks
CREATE TABLE IF NOT EXISTS time_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    teacher_id TEXT NOT NULL,
    start TIMESTAMP WITH TIME ZONE NOT NULL,
    "end" TIMESTAMP WITH TIME ZONE NOT NULL,
    reason TEXT,
    type TEXT NOT NULL CHECK (type IN ('vacation', 'sick', 'other')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_time_blocks_teacher_id ON time_blocks (teacher_id);
CREATE INDEX IF NOT EXISTS idx_time_blocks_dates ON time_blocks (teacher_id, start, "end");

-- Slots
CREATE TABLE IF NOT EXISTS slots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    teacher_id TEXT NOT NULL,
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status TEXT NOT NULL DEFAULT 'free' CHECK (status IN ('free', 'booked', 'cancelled', 'blocked')),
    booking_id UUID,
    template_id UUID REFERENCES availability_templates(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_slots_teacher_id ON slots (teacher_id);
CREATE INDEX IF NOT EXISTS idx_slots_dates ON slots (start, "end");
CREATE INDEX IF NOT EXISTS idx_slots_status ON slots (status);
CREATE INDEX IF NOT EXISTS idx_slots_booking_id ON slots (booking_id);
CREATE INDEX IF NOT EXISTS idx_slots_template_id ON slots (template_id);

-- Bookings
CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slot_id UUID NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    student_id TEXT NOT NULL,
    teacher_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'cancelled')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    cancelled_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_bookings_slot_id ON bookings (slot_id);
CREATE INDEX IF NOT EXISTS idx_bookings_student_id ON bookings (student_id);
CREATE INDEX IF NOT EXISTS idx_bookings_teacher_id ON bookings (teacher_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings (status);

-- Attendance
CREATE TABLE IF NOT EXISTS attendance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('present', 'absent', 'late')),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_attendance_booking_id ON attendance (booking_id);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_availability_templates_updated_at
    BEFORE UPDATE ON availability_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_time_blocks_updated_at
    BEFORE UPDATE ON time_blocks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_slots_updated_at
    BEFORE UPDATE ON slots
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bookings_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_attendance_updated_at
    BEFORE UPDATE ON attendance
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
