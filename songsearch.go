package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// TopNSongsToReturn stores the number of songs worth of lyric data to return.
var TopNSongsToReturn = 10

// NumContextWords stores the number of words around the search word to include.
var NumContextWords = 5

// song stores information about a song.
type song struct {
	artist string
	title  string
	lyrics []string
}

// songUsage stores information about every time
// the given word was used in the specified song.
type songUsage struct {
	songIndex int
	positions []int
}

func makeSongsArray(filename string) []song {
	var songs []song
	csvFile, error := os.Open(filename)
	if error != nil {
		log.Fatal(error)
	}
	r := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, error := r.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		// Skip the first line, which identifies each column.
		if line[0] == "artist" {
			continue
		}
		s := song{artist: line[0], title: line[1], lyrics: strings.Split(line[3], " ")}
		songs = append(songs, s)
	}
	fmt.Printf("Done reading lyric data into memory.\n")
	return songs
}

func makeContextFromWordIndices(songs []song, songIndex int, indices []int) []string {
	s := songs[songIndex]
	lyricsLength := len(s.lyrics)
	var context []string
	for _, wordIndex := range indices {
		// We want to include the number of context words around the word
		// wherever possible. Otherwise, use the beginning and end of the lyrics
		// as bounds.
		var startIndex, endIndex int
		lastIndex := lyricsLength - 1
		if wordIndex < NumContextWords {
			startIndex = 0
			endIndex = wordIndex + NumContextWords
		} else if wordIndex > lastIndex-NumContextWords {
			startIndex = wordIndex - NumContextWords
			endIndex = lyricsLength - 1
		} else {
			startIndex = wordIndex - NumContextWords
			endIndex = wordIndex + NumContextWords
		}
		var wordsBuf bytes.Buffer
		for index := startIndex; index <= endIndex; index++ {
			wordsBuf.WriteString(s.lyrics[index])
			wordsBuf.WriteString(" ")
		}
		context = append(context, wordsBuf.String())
	}
	return context
}

func bubbleSortSongUsages(usages []songUsage) {
	// Given a list of songUsages, we want to order them by how many occurrences
	// are included. Bubble sort it up to where it needs to be. O(n).
	for index := len(usages) - 1; index > 0; index-- {
		if len(usages[index].positions) > len(usages[index-1].positions) {
			temp := usages[index]
			usages[index] = usages[index-1]
			usages[index-1] = temp
		} else {
			// We're in the correct position.
			break
		}
	}
}

func doSearch(songs []song, usages map[string][]songUsage) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter a word to search, or EXIT to exit: ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		if text == "EXIT" {
			break
		}
		usage, ok := usages[text]
		if ok {
			for i, songUsage := range usage {
				fmt.Printf("Result #%d has %d occurrences:\n\n\n", i+1, len(songUsage.positions))
				artist := songs[songUsage.songIndex].artist
				title := songs[songUsage.songIndex].title
				contextStrings := makeContextFromWordIndices(songs, songUsage.songIndex, songUsage.positions)
				for j := range songUsage.positions {
					fmt.Printf("Title: %s\n", title)
					fmt.Printf("Artist: %s\n", artist)
					fmt.Printf("Context: %s\n\n", contextStrings[j])
				}
			}
		} else {
			fmt.Printf("Word not found.\n")
		}
	}
}

func main() {
	start := time.Now()
	songs := makeSongsArray("songdata.csv")
	mostCommonUsages := make(map[string][]songUsage)
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}

	// Process each song.
	for songIndex, song := range songs {
		songWordUsages := make(map[string][]int)

		// Read the song's lyrics, mapping word to every index where that word
		// occurs.
		for wordIndex, word := range song.lyrics {
			normalizedWord := reg.ReplaceAllString(word, "")
			normalizedWord = strings.ToLower(word)
			if strings.Contains(normalizedWord, "oink") {
				fmt.Printf("!!! OINK FOUND: %s", normalizedWord)
			}
			_, ok := songWordUsages[normalizedWord]
			if ok {
				songWordUsages[normalizedWord] = append(songWordUsages[normalizedWord], wordIndex)
			} else {
				songWordUsages[normalizedWord] = []int{wordIndex}
			}
		}

		// Given the map from word to occurrences, check to see if the
		// number of occurrences in this song is globally significant (within
		// the top N.)
		for word, indices := range songWordUsages {
			globalUsages, ok := mostCommonUsages[word]

			// If there's already global occurrence data for this word, check
			// to see if our data is significant enough to be included.
			if ok {
				if len(globalUsages) < TopNSongsToReturn {
					// context := makeContextFromWordIndices(songs, songIndex, indices)
					s := songUsage{songIndex: songIndex, positions: indices}
					globalUsages = append(globalUsages, s)
					bubbleSortSongUsages(globalUsages)
					mostCommonUsages[word] = globalUsages
				} else {
					// If the number of occurrences at position 10 is greater
					// than what we've got here, don't bother saving it. Otherwise
					// save it and insertion sort to get it into correct order.
					if len(globalUsages[TopNSongsToReturn-1].positions) < len(indices) {
						// positions := makepositionsFromWordIndices(songs, songIndex, indices)
						s := songUsage{songIndex: songIndex, positions: indices}
						globalUsages[TopNSongsToReturn-1] = s
						bubbleSortSongUsages(globalUsages)
						mostCommonUsages[word] = globalUsages
					}
				}
			} else {
				// There isn't anything global data for this word yet, so add
				// our data.
				// positions := makepositionsFromWordIndices(songs, songIndex, indices)
				s := songUsage{songIndex: songIndex, positions: indices}
				usage := []songUsage{s}
				mostCommonUsages[word] = usage
			}
		}
	}
	duration := time.Now().Sub(start)
	fmt.Printf("Data structure built. Time: %.3f seconds\n", duration.Seconds())
	doSearch(songs, mostCommonUsages)
	return
}
