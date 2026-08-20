CREATE TABLE campaign_characters (
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    character_id uuid NOT NULL REFERENCES characters(id),
    role text NOT NULL CHECK (role IN ('PRIMARY', 'LISTENER')),
    selected_by uuid NOT NULL REFERENCES internal_users(id),
    selected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, role),
    UNIQUE (campaign_id, character_id)
);
