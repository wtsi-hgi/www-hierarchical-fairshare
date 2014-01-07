package main

/* set CGO_LDFLAGS and CGO_CFLAGS environment variables if LSF is not installed in system dirs */

/*
#cgo LDFLAGS: -lbat -llsf -lnsl -lm
#include "lsf/lsf.h"
#include "lsf/lsbatch.h"
#include <string.h>
#define HFSGO_MAX_ERRMSG_LEN 100

typedef struct {
  char *queue_name;
  struct queueInfoEnt *info_entry;
} hfsgo_queue_info_t;

char __hfsgo_errmsg[HFSGO_MAX_ERRMSG_LEN+1];

char* hfsgo_errmsg() {
  if (lsberrno == LSBE_NO_ERROR) {
    return __hfsgo_errmsg;
  } else {
    return lsb_sysmsg();
  }
}

hfsgo_queue_info_t* hfsgo_init(char *app_name, char *queue_name) {
  if (lsb_init(app_name) != 0) {
    return NULL;
  } 
  hfsgo_queue_info_t *qi = (hfsgo_queue_info_t *)calloc(1, sizeof(hfsgo_queue_info_t));
  if (qi == NULL) {
    strncpy(__hfsgo_errmsg, "hfsgo_init: failed to allocate memory", HFSGO_MAX_ERRMSG_LEN);
  }
  qi->queue_name = strdup(queue_name);
  return qi; 
}

int hfsgo_queue_info(hfsgo_queue_info_t *qi) {
  int num_queues = 1; // will be updated by lsb_queueinfo
  qi->info_entry = lsb_queueinfo(&qi->queue_name, &num_queues, NULL, NULL, 0);
  if (qi->info_entry == NULL || num_queues <= 0) {
    return -1; // failure
  } else {
    return 0; //success
  }
}

struct shareAcctInfoEnt* hfsgo_get_share_acct(hfsgo_queue_info_t *qi, int n) {
  if (n < qi->info_entry->numOfSAccts) {
    struct shareAcctInfoEnt *sai = &(qi->info_entry->shareAccts[n]);
    return(sai);
  } else {
    return(NULL);
  }
}
*/
import "C"
import "fmt"
import "strings"
import "os"
import "unsafe"
import "flag"
import "math"
import "encoding/json"
import "io"
import "net/http"
import "regexp"
import "log"
import "time"

//import "runtime/pprof"

type ShareAccountEntry struct {
	UserGroupPath string
	UserGroupName string
	ParentPath string
	ParentName string
	Shares int
	Priority float32
	NumStartJobs int
	HistCpuTime float32
	NumReserveJobs int
	RunTime int
	ShareAdjustment float32
	NumForwPendJobs int
	RemoteLoad float32
	Flags int
	Children []*ShareAccountEntry
	ChildrenSharesSum int
	NormalisedShares float64 
	OverallNormalisedShares float64
	OverallPriority float64
	OverallPriorityLog float64
}

func LsbQueueInfo(queueName *string) (qi *C.hfsgo_queue_info_t, err error) {
	qi = C.hfsgo_init(C.CString("hfs.go"), C.CString(*queueName))
	if (C.hfsgo_queue_info(qi) != 0) {
		err = fmt.Errorf("LsbQueueInfo hfsgo_init failed: %s", C.GoString(C.hfsgo_errmsg()))
		return 
	}
	return
}

func NewShareAccountEntry(csai *C.struct_shareAcctInfoEnt) *ShareAccountEntry {
	if unsafe.Pointer(csai) == nil {
		return nil
	}
	sae := new(ShareAccountEntry)
	sae.UserGroupPath = C.GoString(csai.shareAcctPath)
	pathSlice := strings.Split(sae.UserGroupPath, "/")
	sae.UserGroupName = pathSlice[len(pathSlice)-1]
	sae.ParentPath = strings.Join(pathSlice[0:len(pathSlice)-1], "/")
	sae.ParentName = pathSlice[len(pathSlice)-2]
	sae.Shares = int(csai.shares)
	sae.Priority = float32(csai.priority)
	sae.NumStartJobs = int(csai.numStartJobs)
	sae.HistCpuTime = float32(csai.histCpuTime)
	sae.NumReserveJobs = int(csai.numReserveJobs)
	sae.RunTime = int(csai.runTime)
	sae.ShareAdjustment = float32(csai.shareAdjustment)
	sae.NumForwPendJobs = int(csai.numForwPendJobs)
	sae.RemoteLoad = float32(csai.remoteLoad)
	sae.Flags = int(csai.flags)
	return sae
}

func LsbShareAccounts(qi *C.hfsgo_queue_info_t) (shareAccountRoot *ShareAccountEntry, err error) {
	queueName := C.GoString(qi.queue_name)
	shareAccountRoot = new(ShareAccountEntry)
	shareAccountRoot.UserGroupPath = queueName
	shareAccountRoot.UserGroupName = queueName
	shareAccountRoot.Shares = 1
	shareAccounts := make(map[string]*ShareAccountEntry)
	for i := C.int(0); i < qi.info_entry.numOfSAccts; i++ {
		sae := NewShareAccountEntry(C.hfsgo_get_share_acct(qi, i))
		shareAccounts[sae.UserGroupPath] = sae
	}
	if len(shareAccounts) != int(qi.info_entry.numOfSAccts) {
		err = fmt.Errorf("Found %v share accounts but were expecting %v\n", len(shareAccounts), int(qi.info_entry.numOfSAccts))
		return
	}
	// loop through shareAccounts map to connect children to parents and sum shares
	for _, sa := range shareAccounts {
		parent, ok := shareAccounts[sa.ParentPath]
		if !ok {
			if sa.ParentPath == queueName {
//				fmt.Printf("Have root node child %v\n", sa.UserGroupPath)
				// connect to root node
				parent = shareAccountRoot
			} else {
				err = fmt.Errorf("Could not find parent ParentPath=%v for UserGroupPath=%v\n", sa.ParentPath, sa.UserGroupPath)
				return 
			}
		}
//		fmt.Printf("Have child %v of %v\n", sa.UserGroupPath, parent.UserGroupPath)
		siblings := parent.Children
		parent.Children = append(siblings, sa)
		parent.ChildrenSharesSum += sa.Shares

//		fmt.Printf("ParentName=%v ParentPath=%v UserGroupName=%v UserGroupPath=%v\n", sa.ParentName, sa.ParentPath, sa.UserGroupName, sa.UserGroupPath)
	}
	return
}

type shareAccountChildrenVisitor func(parent *ShareAccountEntry, child *ShareAccountEntry)

func traverseAccountChildren(sa *ShareAccountEntry, visit shareAccountChildrenVisitor) (err error) {
//	fmt.Printf("traverseAccountChildren: traversing %v\n", sa.UserGroupName)
	for _, child := range sa.Children {
		visit(sa, child)
		// recurse
		err = traverseAccountChildren(child, visit)
		if err != nil {
			return
		}
	}
	return
}

func addNormalisedAndOverall(parent *ShareAccountEntry, child *ShareAccountEntry) {
//	fmt.Printf("addNormalisedAndOverall: child %v parent %v\n", child.UserGroupName, parent.UserGroupName)
	child.NormalisedShares = float64(child.Shares) / float64(parent.ChildrenSharesSum)
	child.OverallNormalisedShares = float64(child.NormalisedShares) * parent.OverallNormalisedShares
	child.OverallPriority = float64(child.Priority) * parent.OverallPriority
	child.OverallPriorityLog = math.Log10(float64(child.Priority)) + parent.OverallPriorityLog
}

func writeFairShare(jsonout io.Writer, queue *string) error {
//	fmt.Println("Getting queue info...")
	qi, err := LsbQueueInfo(queue)
	if err != nil {
		return err
	}
	
	timestamp := time.Now().Format(time.RFC3339)
	
	fairshareQueueStr := C.GoString(qi.info_entry.fairshareQueues)
	noqueues, matcherr := regexp.MatchString("^[[:space:]]*$", fairshareQueueStr)
	if matcherr != nil {
		return matcherr
	}
	if noqueues {
		fairshareQueueStr = fmt.Sprintf("%s", *queue)
	}

	fairshareQueues := strings.Split(fairshareQueueStr, " ")

//	shareAccountCount := qi.info_entry.numOfSAccts

	shareAccounts, err := LsbShareAccounts(qi)
	if err != nil {
		return err
	}
//	fmt.Printf("Have share account info: %+v\n", shareAccounts)

	// set root node normalised shares and overall shares
	shareAccounts.NormalisedShares = 1.0
	shareAccounts.OverallNormalisedShares = 1.0
	shareAccounts.OverallPriority = 1.0
	shareAccounts.OverallPriorityLog = math.Log10(1.0)

	// descend through the tree and calculate normalised shares
	// and overall shares and priority
	traverseAccountChildren(shareAccounts, addNormalisedAndOverall)

	var output = make(map[string]interface{})
	output["queues"] = fairshareQueues
	output["shares"] = shareAccounts
	output["timestamp"] = timestamp

	enc := json.NewEncoder(jsonout)
	err = enc.Encode(output)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return err
}

func FairShareServer(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile("^/(.*)$")
	queue := re.FindStringSubmatch(req.URL.Path)[1]
	fmt.Fprintf(os.Stderr, "Getting fairshare for queue %s\n", queue)

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	if err := writeFairShare(w, &queue); err != nil {
		noqueue, matcherr := regexp.MatchString("[Nn]o such queue", err.Error())
		if matcherr == nil {
			if noqueue {
				http.Error(w, fmt.Sprintf("No such queue: %v\n", queue), 404)
				return
			}
		} else {
			errmsg := fmt.Sprintf("Error matching string: %v\n", matcherr)
			fmt.Fprintf(os.Stderr, errmsg)
			http.Error(w, errmsg, 500)
			return
		}
		errmsg := fmt.Sprintf("Error getting LSF fairshare data for queue %s: %v\n", queue, matcherr)
		fmt.Fprintf(os.Stderr, errmsg)
		http.Error(w, errmsg, 500)
	}
}


func main() {
	var address = flag.String("address", ":9001", "Address to listen on for HTTP requests")
//	var cert = flag.String("cert", "cert.pem", "TLS certificate file (PEM encoded)")
//	var key = flag.String("key", "key.pem", "TLS key file (PEM encoded)")
	flag.Parse()
	
	http.HandleFunc("/", FairShareServer)
	fmt.Printf("hfs_httpserver about to start listening at %s\n", *address)
//	err := http.ListenAndServeTLS(*address, *cert, *key,  nil)
	err := http.ListenAndServe(*address, nil)
	if err != nil {
//		log.Fatal("ListenAndServeTLS: ", err)
		log.Fatal("ListenAndServe: ", err)
	}
}
