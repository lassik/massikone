CREATE TABLE 'version' (
  'version' integer NOT NULL PRIMARY KEY
);

CREATE TABLE 'setting' (
  'name' varchar(255) NOT NULL PRIMARY KEY,
  'value' varchar(255) NOT NULL
);

CREATE TABLE 'user' (
  'user_id' integer NOT NULL PRIMARY KEY,
  'full_name' varchar(255) NOT NULL,
  'permission_level' integer DEFAULT (0) NOT NULL
);

CREATE TABLE 'user_auth' (
  'user_id' integer NOT NULL REFERENCES 'user',
  'auth_provider' varchar(255) NOT NULL,
  'auth_user_id' varchar(255) NOT NULL,
  PRIMARY KEY ('user_id', 'auth_provider')
);

CREATE TABLE 'period' (
  'period_id' integer NOT NULL PRIMARY KEY,
  'start_date' varchar(255) NULL,
  'end_date' varchar(255) NULL
);

CREATE TABLE 'period_account' (
  'period_id' integer NOT NULL REFERENCES 'period',
  'account_id' integer NOT NULL,
  'account_type' integer NOT NULL,
  'title' varchar(255) NOT NULL,
  'starting_balance_cents' integer DEFAULT (0) NOT NULL,
  'nesting_level' integer DEFAULT (0) NOT NULL,
  PRIMARY KEY ('period_id', 'account_id', 'nesting_level')
);

CREATE TABLE 'bill' (
  'bill_id' integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  'description' varchar(255) DEFAULT ('') NOT NULL,
  'paid_date' varchar(255) NULL,
  'paid_user_id' integer NULL REFERENCES 'user',
  'closed_date' varchar(255) NULL,
  'closed_user_id' integer NULL REFERENCES 'user',
  'created_date' varchar(255) NOT NULL,
  'closed_type' varchar(255) NULL
);

CREATE TABLE 'bill_entry' (
  'bill_id' integer NOT NULL REFERENCES 'bill',
  'row_number' integer NOT NULL,
  'account_id' integer NOT NULL,
  'debit' Boolean NOT NULL,
  'unit_count' integer DEFAULT (1) NOT NULL,
  'unit_cost_cents' integer NOT NULL,
  'description' varchar(255) NOT NULL,
  PRIMARY KEY ('bill_id', 'row_number')
);

CREATE TABLE 'image' (
  'image_id' varchar(255) NOT NULL PRIMARY KEY,
  'image_data' blob NOT NULL
);

CREATE TABLE 'bill_image' (
  'bill_id' integer NOT NULL REFERENCES 'bill',
  'bill_image_num' integer NOT NULL,
  'image_id' varchar(255) NOT NULL REFERENCES 'image',
  PRIMARY KEY ('bill_id', 'bill_image_num')
);

INSERT INTO setting values ("OrgFullName", "");
INSERT INTO setting values ("OrgShortName", "");

INSERT INTO version (version) values (1);
