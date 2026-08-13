-- Les sociétés conventionnées existantes, fournies par la Direction.
--
-- En migration et non en script : c'est une donnée de référence, elle doit exister à
-- l'identique dans chaque environnement. ON CONFLICT DO NOTHING la rend rejouable — et
-- absorbe les doublons de la liste d'origine, où SNEF et DIETSMAN figuraient deux fois
-- (positions 25/45 et 16/46). L'index unique de la 031 porte sur LOWER(TRIM(name)) :
-- c'est lui qui les ramène à une seule fiche chacune.
--
-- Volontairement absente : « ICS INTER CONTINENTAL DES C... », dont le nom était tronqué
-- dans la liste transmise. Inscrire une raison sociale incomplète dans la table qui sert
-- justement à normaliser les noms aurait été contradictoire ; elle s'ajoute depuis l'écran
-- Sociétés de la Direction dès que son intitulé exact est connu.
INSERT INTO societes (name) VALUES
    ('PARTICULIER(E)'),
    ('CONGO OILFIELD'),
    ('BCFI'),
    ('AGL'),
    ('GNCAC'),
    ('DIXSTONE OPERATIONS'),
    ('ERNST & YOUNG CONGO'),
    ('BANK OF AFRICA BOA'),
    ('BAKER HUGHES'),
    ('CENTRALE ELECTRIQUE DU CONGO'),
    ('SERVIZI ENERGIA ITALIA SPA'),
    ('CONSEIL CONGOLAIS DES CHARGES'),
    ('MAROCO CONSULTING'),
    ('ILOGS'),
    ('UCAC-ICAM'),
    ('BOURBON'),
    ('DIETSMAN'),
    ('HALLIBURTON'),
    ('SNPC'),
    ('AIRTEL'),
    ('EMPLOI SERVICE'),
    ('SAT CONGO'),
    ('BSCA'),
    ('BANQUE POSTALE'),
    ('CFAO'),
    ('SNEF'),
    ('WEATHERFORDS'),
    ('ES-CO'),
    ('SAS CONGO'),
    ('SGMP'),
    ('EXPRO'),
    ('BEAC'),
    ('TOTAL ENERGIES'),
    ('ECOBANK'),
    ('CREDIT DU CONGO'),
    ('IES ROW CONGO'),
    ('PONTICECL CONGO'),
    ('CORAF'),
    ('SCHLUMBERGER LOGELO'),
    ('WTW'),
    ('BUREAU VERITAS'),
    ('WING WAH'),
    ('GROUPE FOBERD'),
    ('CONGO TERMINAL'),
    ('COFINA'),
    ('BOSCONGO'),
    ('CMA.CG'),
    ('DANGOTE'),
    ('PONTICELI'),
    ('PERENCO'),
    ('LOANGO ENV'),
    ('MI OVERSEAS LIMITED-GS'),
    ('RENCO'),
    ('FIDAFRICA'),
    ('CARTAGO'),
    ('PIC'),
    ('AMMAT GLOBAL'),
    ('SFP'),
    ('AMERAUDE'),
    ('CONGO REP')
ON CONFLICT DO NOTHING;
