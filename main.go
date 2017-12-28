package main

/* ----------------------------
    Librerías
-----------------------------*/
import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/kennygrant/sanitize"
)

/* ----------------------------
    Declaraciones globales
-----------------------------*/

//const pathToVisualizacionesPy = "/usr/local/go/workspace/servidor/visualizaciones.py"
const pathToVisualizacionesPy = "/Users/roberto/Google Drive/Documents/Programming/DataScience/PythonProjects/TwitterAnalyticsPythonScript/visualizaciones.py"

//Estructuras
type serverRequestHandler struct{}

//declaro estructuras para reconocer JSON
//recordar que para Go
//los nombres tienen que empezar con Mayúscula

type pyResultMainImages struct {
	Users    string `json:"users"`
	Words    string `json:"words"`
	Hashtags string `json:"hashtags"`
}

type pyResultsImageTweets struct {
	Images []string `json:"images"`
	URL    string   `json:"url"`
}

type pyResultMedia struct {
	TweetImages []pyResultsImageTweets `json:"tweet_images"`
	SharedUrls  []string               `json:"shared_urls"`
}

type PyJSONResponse struct {
	Images pyResultMainImages `json:"images"`
	Media  pyResultMedia      `json:"media"`
}

/* ----------------------------
    Funciones
-----------------------------*/

//Convertir JSON retornado por Python a un array
func parsePyJSON(body []byte) (*PyJSONResponse, error) {
	var pyResponse = new(PyJSONResponse)
	err := json.Unmarshal(body, &pyResponse)
	return pyResponse, err
}

//Atender requests
func (h serverRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//Proceso las llamadas a los diferentes métodos.
	switch r.Method {
	//Si es un GET
	case "GET":
		//Servir página principal
		if r.URL.Path == "/" {
			//envío headers al navegador para decirle que va un html
			w.Header().Set("Content-Type", "text/html")
			//leo el template para el root
			indexHTMLFile, err := ioutil.ReadFile("index.html")
			//si ocurre un error
			if err != nil {
				//que explote
				fmt.Println("Error reading index html template file:", err)
			}
			//mando el html al navegador después
			//de type-castearlo a un byte array
			w.Write([]byte(indexHTMLFile))
			//Si el request es para Twitter Data Analisis
		} else if r.URL.Path == "/tda" || r.URL.Path == "/tda/" {
			//leo el template
			searchHTMLFile, err := ioutil.ReadFile("tda_template_search.html")
			//si ocurre un error
			if err != nil {
				//que explote
				fmt.Println("Error reading search html template file:", err)
			}
			//declaro variable string
			var replace_by string
			//si encuentro error 1 en url
			if r.URL.Query().Get("err") == "1" {
				//significa que no se econtraron resultados
				replace_by = "<div class=\"alert alert-warning\"><strong>Alert: </strong> No results found.</div>"
			} else {
				//entonces no hubo errores
				replace_by = ""
			}
			//remplazo el placeholder de mensajes con lo que haya quedado
			replace_messages := bytes.Replace(searchHTMLFile, []byte("{{messages}}"), []byte(replace_by), -1)
			//envío headers al navegador para decirle que va un html
			w.Header().Set("Content-Type", "text/html")
			//mando el html al navegador después
			//de type-castearlo a un byte array
			w.Write([]byte(replace_messages))

			//Si el request es desconocido
		} else {
			//mostar error 404 not found
			http.NotFound(w, r)
			//retorno
		}
		break
	//Si es un POST
	case "POST":
		//Le digo a Go que procese los datos del form
		err := r.ParseForm()
		//si ocurre un error
		if err != nil {
			//que reviente todo
			fmt.Println("Error pasring form:", err)
		}
		//hago un bucle leyendo los datos enviados del form
		for key, value := range r.PostForm {
			//si la llave encontrada es "keywords", que es el input de búsqueda
			if key == "keywords" {
				//usar esta librería (que tomé prestada a kennygrant) para limpiar el contenido un poco
				//uniendo los valores de "value" en el caso de que haya mas de uno (sea un arreglo)
				strValue := sanitize.HTML(strings.Join(value, ""))
				//Si no está vacío
				if strings.TrimSpace(strValue) != "" {

					//Paso a crear los archivos temporales, solo para obtener un nombre único para las imágenes
					//la idea es que en multiple connexiones, el nombre de la imágen que genere python sea única
					//que es lo que hace "TempFile"

					//Primer temporal que contendrá la imágen wordcloud de screen names mas activos
					tmp_img_activeusers, err := ioutil.TempFile(os.TempDir(), "actusrs")
					//si ocurre un erro
					if err != nil {
						//haceme llorar
						fmt.Println("Error creating temp actusrs img file:", err)
					}
					//creo segundo temporal que contendrá imagen wordcloud de palabras mas relevantes
					tmp_img_muwords, err := ioutil.TempFile(os.TempDir(), "muwords")
					if err != nil {
						//haceme llorar
						fmt.Println("Error creating temp muwords img file:", err)
					}

					//creo tercer temporal que contendrá imagen wordcloud de hastags mas relevantes
					tmp_img_hashtags, err := ioutil.TempFile(os.TempDir(), "hashtags")
					if err != nil {
						//haceme llorar
						fmt.Println("Error creating temp hashtags img file:", err)
					}

					//ya con los archivos temporales,
					//leo el path y nombre de los archivos completos
					path_img_activeusers := tmp_img_activeusers.Name()
					path_img_muwords := tmp_img_muwords.Name()
					path_img_hashtags := tmp_img_hashtags.Name()

					//Teniendo los nombres únicos
					//Cierro los archivos y borro antes de
					//llamar a Python
					tmp_img_activeusers.Close()
					tmp_img_muwords.Close()
					tmp_img_hashtags.Close()
					os.Remove(path_img_activeusers)
					os.Remove(path_img_muwords)
					os.Remove(path_img_hashtags)

					//a los nombres únicos, le adjunto extension
					path_img_activeusers += ".png"
					path_img_muwords += ".png"
					path_img_hashtags += ".png"

					//Ejecuto el script con Visualizaciones.py
					//pasándole el keyword, y los nombres de los dos
					//archivos de imágen...
					cmdName := "python3"
					cmdArgs := []string{pathToVisualizacionesPy,
						strValue,
						path_img_activeusers,
						path_img_muwords,
						path_img_hashtags}
					//Instancio el exec
					cmd := exec.Command(cmdName, cmdArgs...)
					//creo un buffer para poder leer del stdout
					//y stderr
					var out bytes.Buffer
					var stderr bytes.Buffer
					//asigno
					cmd.Stdout = &out
					cmd.Stderr = &stderr
					//corro el script
					errorMsg := cmd.Run()
					//si hubo errores
					if errorMsg != nil {
						//muestro que fue lo que pasó
						fmt.Println(fmt.Sprint(errorMsg) + ": " + stderr.String())
					}
					//limpio la salida del resultado de pythoin de saltos de lineas
					outputData := strings.TrimSuffix(out.String(), "\n")

					//Pythong no consiguió ningún tuit?
					if outputData == "No twits found for the specified search keywords" {
						//Redirigir temporal a home con error
						http.Redirect(w, r, "/tda/?err=1", http.StatusSeeOther)
					} else {
						//Leo el archivo template que muestra los resultados en un string
						resultsHTMLFile, err := ioutil.ReadFile("tda_template_results.html")
						//si hubo error
						if err != nil {
							//que me importa, que se joda
							fmt.Println("Error reading results html template:", err)
						}
						//Abro las imágenes
						//cargo la imágen wordcloud de screen names que se creó con python
						//usando el nombre temporal creado anteiormente
						file_img_activeusers, err := os.Open(path_img_activeusers)
						//si hay error
						if err != nil {
							//y bueno, así es la vida
							fmt.Println("Error opening activeusers image:", err)
							//si ya se, tendría que salir y reportar
							//error al usuario y demás.
							//pero ahora not engo ganas
						}
						//creo bufer para leer la imágen binaria
						fActiveusersInfo, _ := file_img_activeusers.Stat()
						var fActiveusersFileSize int64 = fActiveusersInfo.Size()
						fActiveusersInfoBuf := make([]byte, fActiveusersFileSize)
						//cargo la imagen en el bufer
						fActiveusersReader := bufio.NewReader(file_img_activeusers)
						fActiveusersReader.Read(fActiveusersInfoBuf)
						//convierto bufer a un string base64
						fActiveusersBase64Str := base64.StdEncoding.EncodeToString(fActiveusersInfoBuf)
						//cierro el binario y borro el archivo de imágen
						file_img_activeusers.Close()
						os.Remove(path_img_activeusers)

						//cargo la imágen wordcloud de las palabras mas usadas que se creó con python
						//usando el nombre temporal creado anteiormente
						file_img_muwords, err := os.Open(path_img_muwords)
						//si da error
						if err != nil {
							//ya esto es un caos
							fmt.Println("Error opening muwords image:", err)
						}
						//creo bufer para leer la imágen binaria
						fMostUsedWordsInfo, _ := file_img_muwords.Stat()
						var fMostUsedWordsFileSize int64 = fMostUsedWordsInfo.Size()
						fMostUsedWordsInfoBuf := make([]byte, fMostUsedWordsFileSize)
						//cargo imagen en bufer
						fUsedWordsReadere := bufio.NewReader(file_img_muwords)
						fUsedWordsReadere.Read(fMostUsedWordsInfoBuf)
						//convierto el bufer a un string encodeado a base64
						fMostUsedWordsBase64Str := base64.StdEncoding.EncodeToString(fMostUsedWordsInfoBuf)
						//cierro y borro archivos para que no queden en el servidor
						file_img_muwords.Close()
						os.Remove(path_img_muwords)

						//cargo la imágen wordcloud de los hashtags mas relevantes que se creó con python
						//usando el nombre temporal creado anteiormente
						file_img_hashtags, err := os.Open(path_img_hashtags)
						//si da error
						if err != nil {
							//que me importa
							fmt.Println("Error opening hashtags image:", err)
						}
						//creo bufer para leer la imágen binaria
						fHashTagsInfo, _ := file_img_hashtags.Stat()
						var fHashTagsFileSize int64 = fHashTagsInfo.Size()
						fHashTagsInfoBuf := make([]byte, fHashTagsFileSize)
						//cargo imagen en bufer
						fHashTagsReadere := bufio.NewReader(file_img_hashtags)
						fHashTagsReadere.Read(fHashTagsInfoBuf)
						//convierto el bufer a un string encodeado a base64
						fHashTagsBase64Str := base64.StdEncoding.EncodeToString(fHashTagsInfoBuf)
						//cierro y borro archivos para que no queden en el servidor
						file_img_hashtags.Close()
						os.Remove(path_img_hashtags)

						//A continuación remplazo las marcas en el template por el contenido del string base65
						//remplazo marca con imagen de activeusers
						replace_activeusers := bytes.Replace(resultsHTMLFile, []byte("{{img_activeusers}}"), []byte(fActiveusersBase64Str), -1)
						//remplazo marca con imagen de most used words
						replace_muwords := bytes.Replace(replace_activeusers, []byte("{{img_muwords}}"), []byte(fMostUsedWordsBase64Str), -1)
						//remplazo marca con imagen de HashTags mas relevantes
						replace_hashtags := bytes.Replace(replace_muwords, []byte("{{img_hashtags}}"), []byte(fHashTagsBase64Str), -1)

						//genero imágenes de tuits
						var tweets_image_gallery_html string
						//parseo json
						resultData, err := parsePyJSON([]byte(outputData))
						if err != nil {
							//no funcó
							fmt.Println("Error parsing JSON:", err)
						} else {
							//recorro las entradas de  imágenes+url de los tuits
							for _, imagesAndURLS := range resultData.Media.TweetImages {
								//recorrro las imágenes encontradas para generar html
								for _, imageURL := range imagesAndURLS.Images {
									//agrego html con link a imágen y url para visitar el tuit
									tweets_image_gallery_html += `
									<div class="col-lg-3 col-md-4 col-xs-6">
										<a href="` + imageURL + `" class="d-block mb-4 h-100" data-fancybox="gallery" data-caption="<a href='` + imagesAndURLS.URL + `' target='_blank'>Click/Tap Here To View Tweet</a>">
											<img class="img-fluid img-thumbnail" src="` + imageURL + `" alt="">
										</a>
									</div>
									`
								}
							}
						}
						//remplazo marca con html para galería de imágenes de tuits
						final_result := bytes.Replace(replace_hashtags, []byte("{{tweets_image_gallery}}"), []byte(tweets_image_gallery_html), -1)

						//envío headers al navegador para decirle que va un html
						w.Header().Set("Content-Type", "text/html")
						//Y por último envío el contenido HTML
						w.Write(final_result)
					}
				} else {
					//en caso de haber estado vacío el value de keyword
					//hago una redirección a TDA nuevamente
					//Si, neceita un reporte de error y eso, pero ahora no tengo ganas
					http.Redirect(w, r, "/tda/", 301)
				}
				break
			}
		}
		break
	default:
		//Si, tendría que haber un error, pero, otro día...
	}
	return
}

/* ----------------------------
Funcion principal (main)
-----------------------------*/
func main() {
	//creo escucho en el puerto 9090 y
	//le paso el serverRequestHandler para manejar los requests
	err := http.ListenAndServe(":9090", serverRequestHandler{})
	//logear error si lo hubo al terminarx
	log.Fatal(err)
	//chau...
}
