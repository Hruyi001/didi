CREATE TABLE IF NOT EXISTS accounts (
  id VARCHAR(64) PRIMARY KEY,
  phone VARCHAR(32) NOT NULL UNIQUE,
  role VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS passenger_profiles (
  id VARCHAR(64) PRIMARY KEY,
  account_id VARCHAR(64) NOT NULL,
  nickname VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS driver_profiles (
  id VARCHAR(64) PRIMARY KEY,
  account_id VARCHAR(64) NOT NULL,
  name VARCHAR(64) NOT NULL,
  phone VARCHAR(32) NOT NULL,
  audit_state VARCHAR(32) NOT NULL,
  work_status VARCHAR(32) NOT NULL,
  accepted_count INT NOT NULL DEFAULT 0,
  rejected_count INT NOT NULL DEFAULT 0,
  timeout_count INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS vehicles (
  id VARCHAR(64) PRIMARY KEY,
  driver_id VARCHAR(64) NOT NULL,
  plate_no VARCHAR(32) NOT NULL,
  model VARCHAR(64) NOT NULL,
  color VARCHAR(32) NOT NULL,
  audit_state VARCHAR(32) NOT NULL,
  UNIQUE KEY uk_vehicles_driver (driver_id)
);

CREATE TABLE IF NOT EXISTS ride_orders (
  id VARCHAR(64) PRIMARY KEY,
  passenger_id VARCHAR(64) NOT NULL,
  driver_id VARCHAR(64),
  pickup_name VARCHAR(255) NOT NULL,
  pickup_lng DECIMAL(10, 6) NOT NULL,
  pickup_lat DECIMAL(10, 6) NOT NULL,
  dropoff_name VARCHAR(255) NOT NULL,
  dropoff_lng DECIMAL(10, 6) NOT NULL,
  dropoff_lat DECIMAL(10, 6) NOT NULL,
  status VARCHAR(32) NOT NULL,
  payment_status VARCHAR(32) NOT NULL,
  review_status VARCHAR(32) NOT NULL,
  estimated_distance DECIMAL(10, 2) NOT NULL,
  estimated_duration INT NOT NULL,
  estimated_amount BIGINT NOT NULL,
  final_amount BIGINT NOT NULL DEFAULT 0,
  cancel_reason VARCHAR(255),
  version BIGINT NOT NULL,
  created_at DATETIME NOT NULL,
  accepted_at DATETIME,
  arrived_at DATETIME,
  started_at DATETIME,
  ended_at DATETIME,
  paid_at DATETIME,
  INDEX idx_ride_orders_passenger (passenger_id),
  INDEX idx_ride_orders_driver (driver_id),
  INDEX idx_ride_orders_status (status)
);

CREATE TABLE IF NOT EXISTS dispatch_tasks (
  id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  candidate_count INT NOT NULL,
  current_attempt_no INT NOT NULL,
  max_attempts INT NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS dispatch_attempts (
  id VARCHAR(64) PRIMARY KEY,
  dispatch_task_id VARCHAR(64) NOT NULL,
  order_id VARCHAR(64) NOT NULL,
  driver_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  distance_to_pickup DECIMAL(10, 2) NOT NULL,
  offered_at DATETIME NOT NULL,
  responded_at DATETIME,
  timeout_at DATETIME NOT NULL,
  reject_reason VARCHAR(255),
  sequence_no INT NOT NULL,
  INDEX idx_dispatch_attempts_order (order_id),
  INDEX idx_dispatch_attempts_driver (driver_id)
);

CREATE TABLE IF NOT EXISTS payment_orders (
  id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  amount BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL,
  paid_at DATETIME,
  UNIQUE KEY uk_payment_orders_order (order_id)
);

CREATE TABLE IF NOT EXISTS payment_transactions (
  id VARCHAR(64) PRIMARY KEY,
  payment_order_id VARCHAR(64) NOT NULL,
  amount BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL,
  failure_reason VARCHAR(255),
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS reviews (
  id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  score INT NOT NULL,
  content VARCHAR(500) NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_reviews_order (order_id)
);

CREATE TABLE IF NOT EXISTS admin_users (
  id VARCHAR(64) PRIMARY KEY,
  username VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL
);

CREATE TABLE IF NOT EXISTS admin_action_logs (
  id VARCHAR(64) PRIMARY KEY,
  admin_id VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NOT NULL,
  target_id VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  before_state JSON,
  after_state JSON,
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS driver_locations (
  driver_id VARCHAR(64) PRIMARY KEY,
  lng DECIMAL(10, 6) NOT NULL,
  lat DECIMAL(10, 6) NOT NULL,
  speed_kph INT NOT NULL,
  updated_at DATETIME NOT NULL
);
