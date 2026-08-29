# Module Inventory

Ce module est propriétaire du stock, des emplacements, des montures et des mouvements de stock.

Règles de frontière:
- Inventory est le seul module qui modifie l'état physique du stock ;
- Reception orchestre l'entrée de montures, mais ne décide pas en dehors de son workflow ;
- Identity n'est pas un module de stock et ne doit pas dépendre de l'inventaire.
