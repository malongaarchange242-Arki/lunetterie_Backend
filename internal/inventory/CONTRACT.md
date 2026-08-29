# Contrat du module Inventory

Ce document fixe la frontiere du module `inventory` dans le monolithe modulaire. Il doit rester valable lorsque le module deviendra `inventory-service`.

## Responsabilite

`inventory` est proprietaire de l'etat physique et commercial immediat des montures :

- montures physiques (`glasses`) ;
- references de montures et attributs catalogue utilises par le stock ;
- emplacements de stockage (`storage_locations`) ;
- capacite et hierarchie `VALISE -> CARTON` ;
- disponibilite, reservation technique et statut d'une monture ;
- mouvements et historique de localisation ;
- analyse associee a une monture et URL de sa photo.

Les autres modules ne doivent pas ecrire directement dans ces tables. Dans le monolithe, ils utilisent les services Go du module ; apres extraction, ils utiliseront l'API HTTP de `inventory-service`.

## Invariants metier

1. Chaque monture possede un barcode unique et stable. Un deplacement ne change jamais son identite.
2. Chaque emplacement possede un identifiant technique stable. Son barcode physique est distinct de cet identifiant.
3. Une `VALISE` n'a pas de parent et un `CARTON` a pour parent une `VALISE`.
4. Un carton ne depasse jamais sa capacite lorsqu'une lunette lui est affectee. Une capacite absente signifie qu'aucune limite n'est configuree.
5. Une affectation ou un deplacement valide la station, le parent, la capacite et l'occupation avant de modifier la lunette.
6. Toute modification de localisation cree un mouvement avec l'utilisateur, la source, la destination et l'action.
7. Une monture ne peut etre reservee que si son statut et sa reservation courante l'autorisent. La reservation ne supprime pas l'historique de localisation.
8. Les operations de mutation doivent etre atomiques : l'etat de la monture, l'etat des emplacements et le mouvement associe sont valides ensemble.
9. Les barcodes d'emplacements sont uniques lorsqu'ils sont renseignes. Les prefixes physiques sont `MEU-`, `VAL-`, `CAR-` et `LUN-`.

## Operations publiques

Les noms ci-dessous sont le contrat fonctionnel. Les routes actuelles peuvent encore avoir des noms historiques ; elles devront converger vers ces operations avant l'extraction.

### Montures

- `CreateGlass` : creer une monture, avec barcode, station, reference optionnelle et statut initial.
- `GetGlass` / `GetGlassByBarcode` : consulter une monture et sa localisation complete.
- `ListGlasses` : filtrer par station, statut, emplacement, reference et reservation.
- `UpdateGlassStatus` : appliquer une transition de statut autorisee.
- `ReserveGlass` : reserver une monture disponible pour une reference externe de commande.
- `ReleaseGlassReservation` : liberer une reservation existante.

### Stockage

- `CreateLocation` : creer une valise ou un carton avec validation du parent.
- `GetLocation` : consulter un emplacement.
- `GetLocationTree` : retourner le chemin et les descendants d'un emplacement.
- `GenerateHierarchy` : generer la hiérarchie physique pour une station.
- `FindAvailableCarton` : trouver un carton compatible sans le reserver.
- `AssignGlass` : affecter une lunette a un carton autorise.

### Mouvements

- `MoveGlass` : deplacer une monture entre emplacements ou stations.
- `RecordMovement` : enregistrer un mouvement systeme explicitement motive.
- `ListMovements` : consulter l'historique filtre et pagine.

### Contrats utilises par les autres modules

- Reception demande la creation d'une monture et son affectation initiale.
- Sales demande la disponibilite, la reservation, la liberation ou le changement de statut d'une monture.
- Les transferts demandent un deplacement inter-station et sa cloture.
- AI fournit une analyse ; `inventory` decide comment la rattacher a une monture. AI ne modifie jamais directement l'inventaire.

## Erreurs attendues

Les erreurs d'API devront rester stables et permettre au client de corriger l'action :

- `GLASS_NOT_FOUND`
- `LOCATION_NOT_FOUND`
- `BARCODE_ALREADY_EXISTS`
- `INVALID_LOCATION_PARENT`
- `LOCATION_CAPACITY_EXCEEDED`
- `LOCATION_OCCUPIED`
- `INVALID_STATUS_TRANSITION`
- `GLASS_ALREADY_RESERVED`
- `GLASS_NOT_AVAILABLE`
- `MOVEMENT_CONFLICT`

Le transport HTTP pourra utiliser `404`, `409` ou `422` selon le cas, mais le code fonctionnel ne doit pas dependre du texte du message.

## Evenements futurs

Le module pourra publier, via outbox, les evenements suivants sans modifier son contrat de mutation :

- `GlassCreated`
- `GlassStatusChanged`
- `GlassReserved`
- `GlassMoved`
- `LocationCreated`

RabbitMQ n'est pas requis pendant la phase de monolithe modulaire.

## Etat actuel et trajectoire

- Les IDs actuels sont des `BIGSERIAL`/`int64`. Ils peuvent rester ainsi pendant la modularisation ; le passage aux UUID est une decision d'extraction, pas une precondition immediate.
- Le schema PostgreSQL est encore partage. L'ownership logique doit etre applique maintenant ; la separation de schema (`inventory.*`) viendra avec l'extraction.
- Les routes actuelles sont principalement sous `/api/v1/inventory/*`. Elles constituent une facade temporaire du module.
- Le contrat physique cible est strictement `VALISE -> CARTON -> LUNETTE`; aucun `MEUBLE` ne doit reaparaitre dans les nouveaux services, DTO, handlers et migrations.
- Les handlers de reception, vente et livraison actuellement places sous `internal/inventory/handlers` devront etre reclasses par responsabilite sans changer leur comportement dans la premiere phase.
- La reservation commerciale, la facture et le paiement appartiennent a `sales` ; `inventory` ne conserve que la reservation technique et l'etat de la monture.

## Regle de migration

Toute nouvelle fonctionnalite doit appeler une operation du contrat plutot que d'ajouter une ecriture SQL directe depuis reception, sales ou un workflow. Cette regle est le principal test de preparation a l'extraction.
