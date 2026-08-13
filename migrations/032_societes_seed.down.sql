-- Ne retire que les fiches encore inutilisées. Une société déjà portée par une proforma
-- est protégée par la clé étrangère de la 031 : la supprimer ferait échouer le retour en
-- arrière, ou effacerait le rattachement d'un document émis. PARTICULIER n'est pas
-- concernée, elle vient de la 031.
DELETE FROM societes
WHERE name IN (
    'CONGO OILFIELD', 'BCFI', 'AGL', 'GNCAC', 'DIXSTONE OPERATIONS',
    'ERNST & YOUNG CONGO', 'BANK OF AFRICA BOA', 'BAKER HUGHES',
    'CENTRALE ELECTRIQUE DU CONGO', 'SERVIZI ENERGIA ITALIA SPA',
    'CONSEIL CONGOLAIS DES CHARGES', 'MAROCO CONSULTING', 'ILOGS', 'UCAC-ICAM',
    'BOURBON', 'DIETSMAN', 'HALLIBURTON', 'SNPC', 'AIRTEL', 'EMPLOI SERVICE',
    'SAT CONGO', 'BSCA', 'BANQUE POSTALE', 'CFAO', 'SNEF', 'WEATHERFORDS', 'ES-CO',
    'SAS CONGO', 'SGMP', 'EXPRO', 'BEAC', 'TOTAL ENERGIES', 'ECOBANK',
    'CREDIT DU CONGO', 'IES ROW CONGO', 'PONTICECL CONGO', 'CORAF',
    'SCHLUMBERGER LOGELO', 'WTW', 'BUREAU VERITAS', 'WING WAH', 'GROUPE FOBERD',
    'CONGO TERMINAL', 'COFINA', 'BOSCONGO', 'CMA.CG', 'DANGOTE', 'PONTICELI',
    'PERENCO', 'LOANGO ENV', 'MI OVERSEAS LIMITED-GS', 'RENCO', 'FIDAFRICA',
    'CARTAGO', 'PIC', 'AMMAT GLOBAL', 'SFP', 'AMERAUDE', 'CONGO REP'
)
AND id NOT IN (
    SELECT societe_id FROM proforma_prescriptions WHERE societe_id IS NOT NULL
);
