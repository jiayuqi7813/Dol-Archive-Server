package main

import (
	"context"
	"fmt"
	yaml "gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Minio MinioConfig `yaml:"minio"`
	Login Login       `yaml:"login"`
}

type Login struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		configData, err := os.ReadFile("config.yaml")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		config := &Config{}
		err = yaml.Unmarshal(configData, config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		auth := c.GetHeader("Authorization")
		if auth != config.Login.Username+":"+config.Login.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}

type MinioConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	UseSSL    bool   `yaml:"secure"`
	Bucket    string `yaml:"bucket"`
}

func InitMinio() (*minio.Client, string, error) {
	configData, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}
	config := &Config{}
	err = yaml.Unmarshal(configData, config)
	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}
	//fmt.Println(config)

	minioClient, err := minio.New(
		config.Minio.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(config.Minio.AccessKey, config.Minio.SecretKey, ""),
			Secure: config.Minio.UseSSL,
		})
	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}
	//判断bucket是否存在，不存在则创建
	err = minioClient.MakeBucket(context.Background(), config.Minio.Bucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), config.Minio.Bucket)
		if errBucketExists == nil && exists {
			fmt.Printf("We already own %s\n", config.Minio.Bucket)
		} else {
			log.Fatal(err)
			return nil, "", err
		}
	}

	return minioClient, config.Minio.Bucket, nil
}

func getSaveDetails(c *gin.Context) {
	// Initialize MinIO client
	minioClient, bucketName, err := InitMinio()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//获取bucket中的saveDetails.json文件
	objectName := "saveDetails.json"
	reader, err := minioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	//将saveDetails.json文件返回
	data, err := io.ReadAll(reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func GetSaves(c *gin.Context) {
	// Initialize MinIO client
	minioClient, bucketName, err := InitMinio()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//获取bucket中的saveDetails.json文件
	objectName := "saves.json"
	reader, err := minioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	//将saveDetails.json文件返回
	data, err := io.ReadAll(reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	// Initialize MinIO client
	//endpoint := "23.224.55.82:9001"
	//accessKey := "iGMOtraTP7W4uUO60aKL"
	//secretKey := "fowF9RezSTYUuc9sX3hQkQnJf1lM8e9L7rrWG6JE"
	//useSSL := false
	//
	//minioClient, err := minio.New(
	//	endpoint, &minio.Options{
	//		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
	//		Secure: useSSL,
	//	})
	minioClient, bucketName, err := InitMinio()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	objectName := file.Filename
	// Upload the file to the bucket
	_, err = minioClient.PutObject(context.Background(), bucketName, objectName, src, file.Size, minio.PutObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
}

func main() {
	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	router.Static("img", "./static/img")
	router.StaticFile("/", "static/auth.html")
	router.GET("/dol", AuthRequired(), func(c *gin.Context) {
		c.File("static/dol.html")
	})
	router.POST("/upload", uploadFile)
	router.GET("/saveDetails", getSaveDetails)
	router.GET("/saves", GetSaves)

	err := router.Run(":8080")
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
