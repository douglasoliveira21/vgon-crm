DO $$
DECLARE
    fk RECORD;
    columns_sql TEXT;
    referenced_columns_sql TEXT;
    delete_action TEXT;
BEGIN
    FOR fk IN
        SELECT
            con.oid,
            con.conname,
            con.conrelid,
            con.confrelid,
            con.conkey,
            con.confkey,
            nsp.nspname AS schema_name,
            rel.relname AS table_name,
            bool_and(att.attnotnull = false) AS all_columns_nullable
        FROM pg_constraint con
        JOIN pg_class rel ON rel.oid = con.conrelid
        JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
        JOIN unnest(con.conkey) WITH ORDINALITY AS cols(attnum, ord) ON true
        JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = cols.attnum
        WHERE con.contype = 'f'
          AND con.confrelid = 'public.users'::regclass
          AND nsp.nspname = 'public'
        GROUP BY con.oid, con.conname, con.conrelid, con.confrelid, con.conkey, con.confkey, nsp.nspname, rel.relname
    LOOP
        SELECT string_agg(quote_ident(att.attname), ', ' ORDER BY cols.ord)
        INTO columns_sql
        FROM unnest(fk.conkey) WITH ORDINALITY AS cols(attnum, ord)
        JOIN pg_attribute att ON att.attrelid = fk.conrelid AND att.attnum = cols.attnum;

        SELECT string_agg(quote_ident(att.attname), ', ' ORDER BY cols.ord)
        INTO referenced_columns_sql
        FROM unnest(fk.confkey) WITH ORDINALITY AS cols(attnum, ord)
        JOIN pg_attribute att ON att.attrelid = fk.confrelid AND att.attnum = cols.attnum;

        IF fk.all_columns_nullable THEN
            delete_action := 'ON DELETE SET NULL';
        ELSE
            delete_action := 'ON DELETE CASCADE';
        END IF;

        EXECUTE format(
            'ALTER TABLE %I.%I DROP CONSTRAINT %I',
            fk.schema_name,
            fk.table_name,
            fk.conname
        );

        EXECUTE format(
            'ALTER TABLE %I.%I ADD CONSTRAINT %I FOREIGN KEY (%s) REFERENCES public.users (%s) %s',
            fk.schema_name,
            fk.table_name,
            fk.conname,
            columns_sql,
            referenced_columns_sql,
            delete_action
        );
    END LOOP;
END $$;
