package datastore

import (
	"strconv"
	"strings"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

// indicate max of each page when paginated repo list
const maxRepoPage = 999

// helper type that sort Feed List
type feedHelper []*model.Feed

func (slice feedHelper) Len() int {
	return len(slice)
}

func (slice feedHelper) Less(i, j int) bool {
	return slice[i].Created < slice[j].Created;
}

func (slice feedHelper) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// rebind is a helper function that changes the sql
// bind type from ? to $ for postgres queries.
func rebind(query string) string {
	if meddler.Default != meddler.PostgreSQL {
		return query
	}

	qb := []byte(query)
	// Add space enough for 5 params before we have to allocate
	rqb := make([]byte, 0, len(qb)+5)
	j := 1
	for _, b := range qb {
		switch b {
		case '?':
			rqb = append(rqb, '$')
			for _, b := range strconv.Itoa(j) {
				rqb = append(rqb, byte(b))
			}
			j++
		case '`':
			rqb = append(rqb, ' ')
		default:
			rqb = append(rqb, b)
		}
	}
	return string(rqb)
}

// helper function that calculate pagination
func calculatePagination(total, limit int) (pages int) {
  pages = total / limit
  if total % limit != 0 {
    pages++
  }
  return
}

// helper function that resize list of repo to pagination
func resizeList(listof []*model.RepoLite, page, limit int) []*model.RepoLite {
	var total = len(listof)
	var end = (page * limit) + limit
	if  end > total{
		end = total
	}
	if total > limit{
		return listof[page * limit:end]
	}
	return listof
}

// helper function that converts a simple repsitory list
// to a sql IN statment.
func toList(listof []*model.RepoLite) (string, []interface{}) {
	var size = len(listof)
	var qs = make([]string, size, size)
	var in = make([]interface{}, size, size)
	for i, repo := range listof {
		qs[i] = "?"
		in[i] = repo.FullName
	}
	return strings.Join(qs, ","), in
}

// helper function that converts a simple repository list
// to a sql IN statement compatible with postgres.
func toListPosgres(listof []*model.RepoLite) (string, []interface{}) {
	var size = len(listof)
	var qs = make([]string, size, size)
	var in = make([]interface{}, size, size)
	for i, repo := range listof {
		qs[i] = "$" + strconv.Itoa(i+1)
		in[i] = repo.FullName
	}
	return strings.Join(qs, ","), in
}
