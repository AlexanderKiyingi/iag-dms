-- Document the controlled vocabularies the distribution desk now offers for
-- outlet price list and volume tier. Columns stay free text so historical
-- values remain valid; Create/Patch normalise known labels to these forms.
--
-- price_list:  wholesale | retail | other
-- volume_tier: bronze | silver | gold | platinum | other

COMMENT ON COLUMN dms_outlets.price_list IS
  'Controlled desk vocabulary: wholesale | retail | other (free text retained for legacy rows)';
COMMENT ON COLUMN dms_outlets.volume_tier IS
  'Controlled desk vocabulary: bronze | silver | gold | platinum | other (free text retained for legacy rows)';
