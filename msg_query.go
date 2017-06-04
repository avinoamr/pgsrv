package srv

import (
    "encoding/binary"
)

// QueryText returns the SQL query string from a Query or Parse message
func (m Msg) QueryText() (string, error) {
    if m.Type() != 'Q' {
        return "", Errf("Not a query message: %q", m.Type())
    }

    return string(m[5:]), nil
}

// RowDescriptionMsg is a message indicating that DataRow messages are about to
// be transmitted and delivers their schema (column names/types)
func RowDescriptionMsg(cols []*ep.Column) Msg {
    msg := []byte{'T', /* LEN = */ 0, 0, 0, 0, /* NUM FIELDS = */ 0, 0}
    binary.BigEndian.PutUint16(msg[5:], uint16(len(cols)))

    for _, c := range cols {
        msg = append(msg, []byte(c.Name)...)
        msg = append(msg, 0) // NULL TERMINATED

        msg = append(msg, 0, 0, 0, 0) // object ID of the table; otherwise zero
        msg = append(msg, 0, 0) // attribute number of the column; otherwise zero

        // object ID of the field's data type
        oid := []byte{0,0,0,0}
        binary.BigEndian.PutUint32(oid, uint32(c.Type.Oid()))
        msg = append(msg, oid...)
        msg = append(msg, 0, 0) // data type size
        msg = append(msg, 0, 0, 0, 0) // type modifier
        msg = append(msg, 0, 0) // format code (text = 0, binary = 1)
    }

    // write the length
    binary.BigEndian.PutUint32(msg[1:5], uint32(len(msg) - 1))
    return msg
}

func DataRowMsg(vals []string) Msg {
    msg := []byte{'D', /* LEN = */ 0, 0, 0, 0, /* NUM VALS = */ 0, 0}
    binary.BigEndian.PutUint16(msg[5:], uint16(len(vals)))

    for _, v := range vals {
        b := append(make([]byte, 4), []byte(v)...)
        binary.BigEndian.PutUint32(b[0:4], uint32(len(b) - 4))
        msg = append(msg, b...)
    }

    // write the length
    binary.BigEndian.PutUint32(msg[1:5], uint32(len(msg) - 1))
    return msg
}

func CompleteMsg(tag string) Msg {
    msg := []byte{'C', 0, 0, 0, 0}
    msg = append(msg, []byte(tag)...)
    msg = append(msg, 0) // NULL TERMINATED

    // write the length
    binary.BigEndian.PutUint32(msg[1:5], uint32(len(msg) - 1))
    return msg
}

func ErrMsg(err error) Msg {
    msg := []byte{'E', 0, 0, 0, 0}

    // https://www.postgresql.org/docs/9.3/static/protocol-error-fields.html
    fields := map[string]string{
        "S": "ERROR",
        "C": "XX000",
        "M": err.Error(),
    }

    epErr, ok := err.(*ep.Err)
    if ok {
        if epErr.Hint != "" {
            fields["H"] = epErr.Hint
        }
        if epErr.Code != "" {
            fields["C"] = epErr.Code
        }
    }


    for k, v := range fields {
        msg = append(msg, byte(k[0]))
        msg = append(msg, []byte(v)...)
        msg = append(msg, 0) // NULL TERMINATED
    }

    msg = append(msg, 0) // NULL TERMINATED

    // write the length
    binary.BigEndian.PutUint32(msg[1:5], uint32(len(msg) - 1))
    return msg
}

// ReadyMsg is sent whenever the backend is ready for a new query cycle.
func ReadyMsg() Msg {
    return []byte{'Z', 0, 0, 0, 5, 'I'}
}
