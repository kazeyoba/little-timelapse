package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// --- Gestion de FFmpeg (Téléchargement & Setup) ---

func setupFFmpeg() string {
	localDir := "./ffmpeg_bin"
	var existingExe string

	// Recherche récursive de l'exécutable ffmpeg.exe
	filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() == "ffmpeg.exe" {
			existingExe = path
			return io.EOF
		}
		return nil
	})

	if existingExe != "" {
		return existingExe
	}

	fmt.Println("📦 FFmpeg non trouvé. Téléchargement de la version Windows...")
	// Utilisation de la version "full" pour garantir tous les codecs
	url := "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-full.zip"
	zipFile := "ffmpeg_temp.zip"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("❌ Erreur de connexion :", err)
		return ""
	}
	defer resp.Body.Close()

	out, _ := os.Create(zipFile)
	io.Copy(out, resp.Body)
	out.Close()

	fmt.Println("📂 Extraction en cours...")
	unzip(zipFile, localDir)
	os.Remove(zipFile)

	var foundPath string
	filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, "ffmpeg.exe") {
			foundPath = path
			return io.EOF
		}
		return nil
	})
	return foundPath
}

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		outFile, _ := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		rc, _ := f.Open()
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
	return nil
}

// --- Fonctions de Capture & Détection GPU ---

func hasNvidiaGPU(ffmpegPath string) bool {
	// Vérifie si l'encodeur est listé dans les capacités de FFmpeg
	cmd := exec.Command(ffmpegPath, "-encoders")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "h264_nvenc")
}

func listWebcamsFFmpeg(ffmpegPath string) []string {
	cmd := exec.Command(ffmpegPath, "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	output, _ := cmd.CombinedOutput()

	re := regexp.MustCompile(`"([^"]+)" \(video\)`)
	matches := re.FindAllStringSubmatch(string(output), -1)

	devices := []string{}
	for _, m := range matches {
		devices = append(devices, m[1])
	}
	return devices
}

func captureImage(ffmpegPath, deviceName, outputPath string) error {
	cmd := exec.Command(ffmpegPath, "-y",
		"-f", "dshow",
		"-video_size", "1920x1080",
		"-i", "video="+deviceName,
		"-frames:v", "1",
		"-q:v", "2",
		outputPath,
	)
	return cmd.Run()
}

func getLastFrameIndex(directory string) int {
	files, _ := filepath.Glob(filepath.Join(directory, "img_*.jpg"))
	if len(files) == 0 {
		return 0
	}
	re := regexp.MustCompile(`img_(\d+)\.jpg`)
	var indices []int
	for _, f := range files {
		match := re.FindStringSubmatch(filepath.Base(f))
		if len(match) > 1 {
			idx, _ := strconv.Atoi(match[1])
			indices = append(indices, idx)
		}
	}
	if len(indices) == 0 {
		return 0
	}
	sort.Ints(indices)
	return indices[len(indices)-1] + 1
}

// --- Main ---

func main() {
	projectName := flag.String("projet", "", "Nom du dossier du projet")
	interval := flag.Int("interval", 5, "Intervalle en secondes")
	useFFmpeg := flag.Bool("render", false, "Générer la vidéo à partir des images")
	flag.Parse()

	if *projectName == "" {
		fmt.Println("❌ Erreur : Le flag --projet est requis (ex: --projet mon_timelapse)")
		return
	}

	ffmpegPath := setupFFmpeg()
	if ffmpegPath == "" {
		fmt.Println("❌ Erreur critique : Impossible de configurer FFmpeg.")
		return
	}

	projectDir, _ := filepath.Abs(*projectName)

	if *useFFmpeg {
		// --- MODE RENDU ---
		outputVideo := *projectName + ".mp4"
		inputPattern := filepath.Join(projectDir, "img_%05d.jpg")

		encoder := "libx264"
		if hasNvidiaGPU(ffmpegPath) {
			encoder = "h264_nvenc"
			fmt.Println("🚀 GPU NVIDIA détecté ! Utilisation de NVENC.")
		} else {
			fmt.Println("💻 GPU NVIDIA non détecté. Utilisation du CPU (libx264).")
		}

		args := []string{"-y", "-framerate", "24", "-i", inputPattern}
		if encoder == "h264_nvenc" {
			args = append(args, "-c:v", "h264_nvenc", "-preset", "p4", "-pix_fmt", "yuv420p", outputVideo)
		} else {
			args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-preset", "medium", outputVideo)
		}

		cmd := exec.Command(ffmpegPath, args...)
		cmd.Stderr = os.Stderr // Affiche les logs FFmpeg pour le debug

		fmt.Printf("🎬 Début de l'encodage (%s)...\n", encoder)
		if err := cmd.Run(); err != nil {
			fmt.Printf("\n❌ Erreur lors du rendu : %v\n", err)
		} else {
			fmt.Printf("\n✅ Vidéo créée avec succès : %s\n", outputVideo)
		}
		return
	}

	// --- MODE CAPTURE ---
	os.MkdirAll(projectDir, 0755)
	cams := listWebcamsFFmpeg(ffmpegPath)
	if len(cams) == 0 {
		fmt.Println("❌ Aucune webcam trouvée. Vérifiez vos branchements.")
		return
	}

	fmt.Println("\n--- Webcams détectées ---")
	for i, name := range cams {
		fmt.Printf("[%d] %s\n", i, name)
	}
	fmt.Print("Choisissez l'index de la caméra : ")

	var idx int
	_, scanErr := fmt.Scanln(&idx)
	if scanErr != nil || idx < 0 || idx >= len(cams) {
		fmt.Println("❌ Entrée invalide.")
		return
	}

	selectedCam := cams[idx]
	photoCount := getLastFrameIndex(projectDir)

	fmt.Printf("\n📸 Capture démarrée sur : %s\n", selectedCam)
	fmt.Printf("📂 Dossier : %s\n", projectDir)
	fmt.Println("⌨️  Appuyez sur Ctrl+C pour arrêter.")

	for {
		fileName := fmt.Sprintf("img_%05d.jpg", photoCount)
		fullPath := filepath.Join(projectDir, fileName)

		err := captureImage(ffmpegPath, selectedCam, fullPath)
		if err != nil {
			fmt.Printf("\n⚠️ Erreur lors de la capture : %v\n", err)
		} else {
			fmt.Printf("📸 Photo #%d enregistrée\r", photoCount)
			photoCount++
		}

		time.Sleep(time.Duration(*interval) * time.Second)
	}
}
