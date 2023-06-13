package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var client *s3.Client

func parse_path(s3URL string) (string, string) {
	parts := strings.Split(s3URL, "/")

	bucket := parts[2] + "/"
	prefix := strings.Join(parts[3:], "/")
	return bucket, prefix
}

func parse_file(s3URL string) (string, string) {
	parts := strings.Split(s3URL, "/")
	length := len(parts)
	bucket := strings.Join(parts[2:length-1], "/")
	prefix := parts[length-1]
	return bucket, prefix
}

func s3listhandler(c *gin.Context) {
	if c.Query("path") != "" {
		// path 有数据时，按照input的s3路径查询数据
		path := c.Query("path")
		// 检查path是否合法
		if strings.HasPrefix(path, "s3://") {

			// 判断是file路径还是dir路径
			if strings.HasSuffix(path, "/") {
				// dir
				// list时最多展示100个
				maxKeys := int32(100)

				bucketName, prefix := parse_path(path)

				input := &s3.ListObjectsV2Input{
					Bucket:    aws.String(bucketName),
					Prefix:    aws.String(prefix),
					MaxKeys:   maxKeys,
					Delimiter: aws.String("/"),
				}

				output, err := client.ListObjectsV2(context.TODO(), input)
				if err != nil {
					message := fmt.Sprintf("Failed to list buckets: %v", err)
					c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": message})
				}

				data := make([]map[string]string, 0)

				for _, object := range output.Contents {
					if aws.ToString(object.Key) == prefix {
						continue
					}
					data = append(data, map[string]string{
						"name":           bucketName + aws.ToString(object.Key),
						"zaOwner":        aws.ToString(object.Owner.DisplayName),
						"zbSize":         strconv.FormatInt(object.Size, 10),
						"zdLastModified": aws.ToTime(object.LastModified).String(),
						"zeType":         "file",
					})
				}

				for _, object := range output.CommonPrefixes {
					data = append(data, map[string]string{
						"name":           bucketName + aws.ToString(object.Prefix),
						"zaOwner":        "",
						"zbSize":         "",
						"zdLastModified": "",
						"zeType":         "dir",
					})
				}

				c.HTML(http.StatusOK, "list.html", gin.H{
					"Data":      data,
					"inputPath": path,
				})
			} else {
				// 文件 只展示前1M的内容
				bucketName, prefix := parse_file(path)
				input := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(prefix),
					Range:  aws.String("bytes=0-1048575"), // 设置 Range 以读取前 1048576 字节
				}

				resp, errg := client.GetObject(context.TODO(), input)
				if errg != nil {
					message := fmt.Sprintf("Error reading objec: %v", errg)
					c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": message})
				}

				defer resp.Body.Close()
				content, _ := ioutil.ReadAll(resp.Body)
				contentAsString := string(content)
				c.JSON(200, gin.H{
					"content": contentAsString,
				})

			}

		} else {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "s3路径非法"})
		}

	} else {
		// 默认展示所有Bucket
		input := &s3.ListBucketsInput{}
		output, err := client.ListBuckets(context.TODO(), input)
		if err != nil {
			message := fmt.Sprintf("Failed to list buckets: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": message})

		}
		data := make([]map[string]string, 0)
		for _, bucket := range output.Buckets {
			data = append(data, map[string]string{
				"name":         aws.ToString(bucket.Name) + "/",
				"zatype":       "Bucket",
				"zbcreatetime": aws.ToTime(bucket.CreationDate).String(),
			})
		}

		c.HTML(http.StatusOK, "list.html", gin.H{
			"Data": data,
		})
	}
}

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	accessKey := viper.GetString("s3.aws_access_key_id")
	secretKey := viper.GetString("s3.aws_secret_access_key")
	endpoint := viper.GetString("s3.endpoint")
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		// config.WithRegion("us-west-2"),
		// aws日志，方便调试
		// config.WithClientLogMode(aws.LogResponseWithBody|aws.LogRequestWithBody),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint}, nil
			})),
	)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}
	// Create an Amazon S3 service client
	client = s3.NewFromConfig(cfg)

	gin.SetMode(viper.GetString("gin.app_mode"))

	router := gin.Default()

	// router.Static("/static", "./website/static")
	router.StaticFile("/favicon.ico", "./website/favicon.ico")
	router.LoadHTMLGlob("website/template/*")

	router.GET("/", s3listhandler)

	fmt.Println("start app...")
	router.Run(viper.GetString("gin.host") + ":" + viper.GetString("gin.port"))

}
