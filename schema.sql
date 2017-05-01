CREATE TABLE IF NOT EXISTS "users" (
 "user_id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
 "email" VARCHAR(50),
 "full_name" VARCHAR(50),
 "iban" VARCHAR(50),
 "is_admin" BOOLEAN,
 "user_id_google_oauth2" VARCHAR(50));

CREATE TABLE IF NOT EXISTS "bills" (
 "bill_id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
 "bill_type" VARCHAR(50),
 "image_id" VARCHAR(50),
 "tags" VARCHAR(50),
 "description" VARCHAR(50),
 "unit_count" INTEGER,
 "unit_cost_cents" INTEGER,
 "paid_date" VARCHAR(50),
 "paid_user_id" INTEGER,
 "reimbursed_date" VARCHAR(50),
 "reimbursed_user_id" INTEGER,
 "closed_date" VARCHAR(50),
 "closed_user_id" VARCHAR(50),
 "created_date" VARCHAR(50),
 paid_type text,
 closed_type text);

CREATE TABLE IF NOT EXISTS tags (
 tag text);

CREATE TABLE IF NOT EXISTS bill_tags (
 bill_id integer,
 tag text);

CREATE TABLE IF NOT EXISTS images (
 image_id text,
 image_data blob);
