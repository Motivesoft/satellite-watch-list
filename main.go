package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", satelliteWatchHandler)

	log.Fatal(http.ListenAndServe(":8081", nil))
}

type PageData struct {
	Title   string
	Heading string
	Count   int
	Content template.HTML
	Status  string
}

func satelliteWatchHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("pages/satellite-passes.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ISS, TianGong, HST
	satelliteIds := []int{25544, 48274, 20580}

	var contents string
	status := "OK"

	sightings, err := getVisualPasses(satelliteIds)
	if err != nil {
		status = fmt.Sprintf("Error: %v", err)
	}

	//	modifyHTML(doc, "content", content)

	for _, sighting := range sightings {
		contents += makeInfoTable(sighting)
		contents += "<br/>"
		contents += makePassTable(sighting)
		contents += "<br/>"
	}

	//	modifyHTML(doc, "sightings", data)

	// Prepare the data, including HTML elements
	data := PageData{
		Title:   "Satellite Watcher",
		Heading: "Satellite Watcher",
		Count:   len(sightings),
		Content: template.HTML(contents),
		Status:  status,
	}

	// Execute the template with the data
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeInfoTable(pass VisualPassesStructure) string {
	html := `
    <h2>%s (%d)</h2>
    <ul>
        <li>Transaction count: %d</li>
        <li>Pass count: %d</li>
    </ul>`

	return fmt.Sprintf(html, pass.Info.SatelliteName, pass.Info.SatelliteId, pass.Info.TransactionsCount, len(pass.Passes))
}

func makePassTable(visualPass VisualPassesStructure) string {
	htmlStart := `
    <table> 
        <tr> 
		    <th>Magnitude</th>
		    <th>Duration</th>
		    <th>Start Visibility</th>
		    <th>Start</th>
		    <th>Start Azimuth</th>
		    <th>Start Elevation</th>
		    <th>Max</th>
		    <th>Max Azimuth</th>
		    <th>Max Elevation</th>
		    <th>End</th>
		    <th>End Azimuth</th>
		    <th>End Elevation</th>
       </tr>`
	htmlMiddle := `
        <tr> 
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%.2f&deg; (%s)</td>
            <td>%.2f&deg;</td>
            <td>%s</td>
            <td>%.2f&deg; (%s)</td>
            <td>%.2f&deg;</td>
            <td>%s</td>
            <td>%.2f&deg; (%s)</td>
            <td>%.2f&deg;</td>
        </tr>`
	htmlEnd := `
    </table>`

	htmlContents := ``

	for _, pass := range visualPass.Passes {
		// Don't display mag if it has the 'unknown' magic value
		var mag string
		if pass.Mag == 100000 {
			mag = "-"
		} else {
			mag = fmt.Sprintf("%.2f", pass.Mag)
		}

		htmlContents += fmt.Sprintf(htmlMiddle,
			mag,
			secondsToDuration(int64(pass.Duration)),
			utcSecondsToLocalTime(pass.StartVisibility),
			utcSecondsToLocalTime(pass.StartUTC),
			pass.StartAz,
			pass.StartAzCompass,
			pass.StartEl,
			utcSecondsToLocalTime(pass.MaxUTC),
			pass.MaxAz,
			pass.MaxAzCompass,
			pass.MaxEl,
			utcSecondsToLocalTime(pass.EndUTC),
			pass.EndAz,
			pass.EndAzCompass,
			pass.EndEl)
	}

	return fmt.Sprintf("%s%s%s", htmlStart, htmlContents, htmlEnd)
}
