DROP TRIGGER IF EXISTS update_attendance_updated_at ON attendance;
DROP TRIGGER IF EXISTS update_bookings_updated_at ON bookings;
DROP TRIGGER IF EXISTS update_slots_updated_at ON slots;
DROP TRIGGER IF EXISTS update_time_blocks_updated_at ON time_blocks;
DROP TRIGGER IF EXISTS update_availability_templates_updated_at ON availability_templates;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS attendance;
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS slots;
DROP TABLE IF EXISTS time_blocks;
DROP TABLE IF EXISTS availability_templates;
