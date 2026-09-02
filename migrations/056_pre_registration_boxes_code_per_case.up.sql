-- Les codes carton recommencent a CTN-001 pour chaque valise.
ALTER TABLE pre_registration_boxes
    DROP CONSTRAINT IF EXISTS pre_registration_boxes_code_key;

ALTER TABLE pre_registration_boxes
    ADD CONSTRAINT pre_registration_boxes_case_id_code_key UNIQUE (case_id, code);
