INSERT INTO item_changes (
  item_id,
  user_id,
  dimension_id,
  quantity,
  state,
  recorded_at
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  datetime('now')
);
