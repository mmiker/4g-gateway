package ini

func inSlice(str string, s []string) bool {
	for _, v := range s {
		if str == v {
			return true
		}
	}
	return false
}
