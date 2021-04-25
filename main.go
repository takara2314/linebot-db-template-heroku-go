package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/line/line-bot-sdk-go/linebot"

	_ "github.com/lib/pq"
)

var (
	bot *linebot.Client
	db  *sql.DB
)

const (
	// LINE Botのチャネルシークレットを設定
	YOUR_CHANNEL_SECRET = "XXXXX"
	// LINE Botのチャネルアクセストークンを設定
	YOUR_CHANNEL_ACCESS_TOKEN = "XXXXX"
	// データベースのホスト名
	DB_HOST = "XXXXX"
	// データベースのデータベース名
	DB_DATABASE = "XXXXX"
	// データベースのユーザー名
	DB_USER = "XXXXX"
	// データベースのポート番号
	DB_PORT = "XXXXX"
	// データベースのパスワード
	DB_PASSWORD = "XXXXX"
)

func main() {
	var err error

	// LINE Botに接続
	bot, err = linebot.New(
		YOUR_CHANNEL_SECRET,
		YOUR_CHANNEL_ACCESS_TOKEN,
	)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	// データベースに接続
	db, err = sql.Open(
		"postgres",
		fmt.Sprintf("postgres://%s:%s@%s:%s/%s", DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_DATABASE),
	)
	if err != nil {
		panic(err)
	}
	// プロセス終了時にデータベースを閉じる
	defer db.Close()

	router := gin.Default()
	// https://example.herokuapp.com/callback にアクセスされたら、
	// callbackPOST という変数を呼び出す
	router.POST("/callback", callbackPOST)

	// ポート番号を環境変数から取得
	var port string = os.Getenv("PORT")
	router.Run(":" + port)
}

// https://example.herokuapp.com/callback にアクセスされたら呼び出される関数
func callbackPOST(c *gin.Context) {
	// LINEから送られてきた情報の前処理
	events, err := bot.ParseRequest(c.Request)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			c.Writer.WriteHeader(400)
		} else {
			c.Writer.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		// ここにメッセージの内容による処理を書いていこう

		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			// メッセージの種類が「テキスト」なら
			case *linebot.TextMessage:
				var responseMessage string = ""

				// 「○○ ×× △△」みたいなスペースで区切られた文章が来るかもしれないので、スペース区切りのスライスにする
				var splited []string = strings.Split(message.Text, " ")

				switch splited[0] {
				case "天気記録":
					if len(splited) == 3 {
						// データベースに天気を記録
						var result string = databaseInsert(splited[1], splited[2])

						if result == "OK" {
							responseMessage = "記録しました！"
						} else {
							responseMessage = "エラーが発生しました…"
						}
					} else {
						responseMessage = "「天気記録 [地域] [天気]」という形で送信してください。"
					}

				case "天気教えて":
					if len(splited) == 2 {
						var weather string = databaseSelect(splited[1])

						switch weather {
						case "sunney":
							responseMessage = fmt.Sprintf("%sの天気は晴れです！", splited[1])
						case "cloudy":
							responseMessage = fmt.Sprintf("%sの天気は曇りです！", splited[1])
						case "rainny":
							responseMessage = fmt.Sprintf("%sの天気は雨です！", splited[1])
						case "snowy":
							responseMessage = fmt.Sprintf("%sの天気は雪です！", splited[1])
						case "none":
							responseMessage = fmt.Sprintf("%sの天気はまだ記録されていません…", splited[1])
						default:
							responseMessage = "エラーが発生しました…"
						}
					} else {
						responseMessage = "「天気教えて [地域]」という形で送信してください。"
					}

				default:
					// message.Text という変数にメッセージの内容が入っている
					switch message.Text {
					case "おはようございます":
						responseMessage = "Good morning!"
					case "こんにちは":
						responseMessage = "Good afternoon!"
					case "こんばんは":
						responseMessage = "Good evening!"
					default:
						responseMessage = "その言葉はわかりません。"
					}
				}

				// 返信文を送信
				// responseMessage の中に入っている文を返す
				_, err = bot.ReplyMessage(
					event.ReplyToken,
					linebot.NewTextMessage(responseMessage),
				).Do()
				if err != nil {
					log.Println(err)
					panic(err)
				}
			}
		}
	}
}

// データベースに天気を記録する関数
func databaseInsert(location string, weather string) string {
	// Execメソッドの引数に、C言語のprintfのような感じにSQL文を入れる
	// 変換指定子の代わりに「$番号」を入れよう (番号は1始まり)
	_, err := db.Exec("INSERT INTO weathers VALUES ($1, $2)", location, weather)
	// 処理でエラーが発生したら、「error」と返す
	if err != nil {
		log.Println("エラーが発生しました:", err)
		return "error"
	}

	return "OK"
}

// データベースから天気を取得する関数
func databaseSelect(location string) string {
	// Execメソッドの引数に、C言語のprintfのような感じにSQL文を入れる
	// 変換指定子の代わりに「$番号」を入れよう (番号は1始まり)
	rows, err := db.Query("SELECT weather FROM weathers WHERE location = $1 LIMIT 1", location)
	// 処理でエラーが発生したら、「error」と返す
	if err != nil {
		log.Println("エラーが発生しました:", err)
		return "error"
	}
	defer rows.Close()

	// 最初の時点で次の行がなければ、データがないということ
	if rows.Next() == false {
		return "none"
	}

	// その行の結果を取り出し、天気の変数にあてる
	// rows.Next() しない限り次の行に移動しない
	var weather string
	rows.Scan(&weather)

	return weather
}
