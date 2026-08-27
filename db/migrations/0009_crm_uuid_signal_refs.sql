-- 0009: Remap stored CRM identifiers in dms_signals to uuid
--
-- dms_signals.entity_id holds whatever identifier arrived on the commercial
-- event that produced the signal (see internal/store/commercial_consumer.go,
-- which reads crm_account_id / account_id off crm.deal.won, crm.lead.converted
-- and crm.outlet.synced).
--
-- CRM keyed its rows on prefixed codes minted as "%s-%04d" — ACC-0500, DEAL-0500
-- — until it moved to uuid. Events published after that carry uuid, so without
-- this migration dms_signals would hold two different identifier styles for the
-- same accounts and nothing would tie the old rows to the new ones.
--
-- The remap uses exactly the mapping CRM applied to itself, so a signal recorded
-- against ACC-0500 ends up on the same uuid the account now has.
--
-- entity_id is display and correlation data, not a foreign key, and it carries
-- identifiers from sources other than CRM. Only values in CRM's minted shape are
-- touched: two to four uppercase letters, a hyphen, then digits. Anything else —
-- a uuid, a DMS outlet code, a free-text reference — is left exactly as it is.

DO $remap$
BEGIN
    IF to_regclass('dms_signals') IS NULL THEN
        RETURN;
    END IF;

    UPDATE dms_signals
       SET entity_id = uuid_in(overlay(overlay(md5('iag:crm:' || entity_id)
                           placing '3' from 13) placing '8' from 17)::cstring)::text
     WHERE entity_id ~ '^[A-Z]{2,4}-[0-9]{4,}$';
END
$remap$;
