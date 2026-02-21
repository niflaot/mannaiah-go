DO $$
BEGIN
    IF to_regclass('public.asset_folders') IS NOT NULL THEN
        CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_folders_slug ON asset_folders(slug);
    END IF;
END $$;
