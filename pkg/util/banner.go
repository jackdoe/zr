package util

func BannerLeft(topDashLen int, prefix string, s []string) string {
	out := "┌"
	for i := 0; i < topDashLen; i++ {
		out += "-"
	}
	out += "\n"

	for _, l := range s {
		out += prefix
		out += " "

		out += l
		out += "\n"
	}
	out += "└--"
	out += "\n"
	return out
}
