-- Réserve les montures dès la création d'une liste d'envoi (Direction, écran Stock Général
-- « Envoyer au Stock Général » ou « Envoyer la liste » depuis une session de réception),
-- avant même le dispatch réel par le magasinier.
--
-- Jusqu'ici rien ne bougeait en base à la création de la liste (SendListHandler.Create) :
-- la monture restait EN_STOCK_GENERAL, sélectionnable dans une deuxième liste en parallèle,
-- sans que rien ne le signale ni au front ni en base.
--
-- RESERVEE_ENVOI et non EN_TRANSIT : ce dernier est réservé aux montures réellement parties
-- (transferableStatuses, resolveGlasses dans send_list_dispatch_service.go). Le réutiliser
-- ici les aurait rendues indispatchables au moment du vrai départ — voir le commentaire
-- historique de la 020 sur le carton perdu, qui a justifié cette séparation stricte entre
-- « réservé » et « réellement parti ».
ALTER TABLE glasses DROP CONSTRAINT IF EXISTS glasses_status_check;
ALTER TABLE glasses ADD CONSTRAINT glasses_status_check CHECK (status IN (
    'RECU_FOURNISSEUR', 'EN_STOCK_GENERAL', 'RESERVEE_ENVOI', 'EN_TRANSIT',
    'EN_STOCK_SOUS_STATION', 'EN_PRESENTOIR', 'EN_CAISSE', 'RESERVEE',
    'EN_LABORATOIRE', 'PRETE_A_LIVRER', 'LIVREE', 'VENDUE',
    'PERDUE', 'CASSEE', 'RETOURNEE'
));
