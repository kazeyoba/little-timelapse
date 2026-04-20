# 📸 Timelapse Tool

Un utilitaire léger en Go qui transforme votre webcam en caméra de timelapse. Il gère automatiquement le téléchargement de FFmpeg et permet de compiler les photos en vidéo MP4.

# 🚀 Guide de démarrage rapide (Windows)

Pas besoin d'installer Go ou FFmpeg manuellement, suivez simplement ces étapes :

## 1. Téléchargement

1. Allez dans l'onglet **[Releases](https://github.com/kazeyoba/little-timelapse/releases)** de ce projet.
2. Téléchargez le fichier `timelapse-tool.exe`.

## 2. Lancement

1. Placez le `.exe` dans un dossier dédié (ex: `C:\Scripts\Timelapse`).
2. Ouvrez un **Terminal** (Faites `Shift + Clic droit` dans le dossier -> "Ouvrir dans le terminal" ou "Ouvrir une fenêtre PowerShell").
3. Lancez la capture avec la commande suivante :

   ```powershell
   ./timelapse-tool.exe --projet mon_timelapse --interval 10
   ```

## 3. Utilisation

* **Choix de la caméra** : Le script listera vos caméras. Tapez le chiffre correspondant (ex: `0`) et validez avec `Entrée`.
* **Capture** : Le script prendra une photo toutes les X secondes.
* **Arrêter** : Faites `Ctrl + C` dans le terminal pour stopper.

# 🛠️ Options de ligne de commande

| Flag | Description | Défaut |
| :--- | :--- | :--- |
| `--projet` | Nom du dossier où stocker les images | **Requis** |
| `--interval`| Temps entre chaque photo (secondes) | `5` |
| `--render`  | Compile les images existantes en vidéo MP4 | `false` |

## Exemples :

**Prendre une photo toutes les minutes :**

```bash
./timelapse-tool.exe --projet jardin --interval 60
```

**Générer la vidéo finale :**

```bash
./timelapse-tool.exe --projet jardin --render
```

# ⚙️ Comment ça marche ?

1. **Auto-Setup** : Au premier lancement, le programme télécharge une version portable de FFmpeg dans le dossier `ffmpeg_bin`.
2. **Capture** : Utilise l'API `dshow` (DirectShow) de Windows pour capturer des images en haute résolution (1920x1080).
3. **Reprise** : Si vous relancez le script sur un projet existant, il détecte automatiquement le dernier numéro d'image pour ne pas écraser vos fichiers.

# 🏗️ Compilation (Pour les développeurs)

Si vous souhaitez modifier le code et compiler vous-même :

1. Installez [Go](https://go.dev/).
2. Clonez le repo.
3. Compilez :
   ```bash
   go build -o timelapse.exe main.go
   ```

---
*Note : Pour le mode `--render`, le script utilise actuellement l'encodeur matériel NVIDIA (`h264_nvenc`). Si vous n'avez pas de GPU NVIDIA, modifiez la ligne correspondante dans le code par `libx264`.*
```

## Quelques conseils pour ton code :
1. **Sécurité des erreurs** : Dans ton code, tu utilises souvent `_` pour ignorer les erreurs (notamment dans `unzip` et `os.Create`). Pour un outil de production, il vaut mieux vérifier si l'écriture sur disque échoue (disque plein, droits admin requis).
2. **Encodeur Vidéo** : Comme indiqué dans le README, `h264_nvenc` est génial mais spécifique aux cartes NVIDIA. Si tu veux que ton outil soit universel, tu pourrais ajouter un flag `--gpu` ou utiliser `libx264` (CPU) par défaut qui fonctionne partout.