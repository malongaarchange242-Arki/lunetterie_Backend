# Module Reception

Ce module orchestre la réception fournisseur et la mise en stock des montures.

Règles de frontière:
- il dépend du module Inventory pour les montures, les emplacements et les règles de stock ;
- il ne doit pas réécrire la logique commerciale de stockage ailleurs que dans le workflow de réception ;
- l'IA reste un service séparé, appelé via un adaptateur externe ou un appel HTTP dédié.
