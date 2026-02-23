package networking

func ParsePtyRequest(payload []byte) (width, height int) {
	if len(payload) < 4 {
		return 80, 24
	}
	termLen := int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	offset := 4 + termLen
	if len(payload) < offset+8 {
		return 80, 24
	}
	w := int(payload[offset])<<24 | int(payload[offset+1])<<16 | int(payload[offset+2])<<8 | int(payload[offset+3])
	h := int(payload[offset+4])<<24 | int(payload[offset+5])<<16 | int(payload[offset+6])<<8 | int(payload[offset+7])
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}
	return w, h
}

func ParseWindowChange(payload []byte) (width, height int) {
	if len(payload) < 8 {
		return 80, 24
	}
	w := int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	h := int(payload[4])<<24 | int(payload[5])<<16 | int(payload[6])<<8 | int(payload[7])
	return w, h
}
