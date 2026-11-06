package blnotifications

// textSet holds every sentence used in the emails, in one language.
// Placeholders are filled with fmt.Sprintf in the order shown in comments.
type textSet struct {
	ourTeam      string
	hi           string // name
	hiCompany    string
	thanks       string
	whatsappLine string
	manageLink   string
	viewOrder    string
	allDay       string
	reasonLabel  string

	labelService, labelDate, labelTime, labelCustomer, labelPhone, labelEmail,
	labelComments, labelReference, labelStatus, labelTotal, labelItems,
	labelDelivery, labelAddress, labelNotes string

	waAboutBooking string // reference
	waAboutOrder   string // reference

	// Appointments — customer
	apptRequestedSubject, apptRequestedBody   string // company | service, date, time, company
	apptCreatedSubject, apptCreatedBody       string // company | service, date, time
	apptApprovedSubject, apptApprovedBody     string // company | service, date, time
	apptRejectedSubject, apptRejectedBody     string // company | service, date, time
	apptDoneSubject, apptDoneBody             string // company | service, date
	apptCancelledSubject, apptCancelledBody   string // company | service, date, time
	apptReopenedSubject, apptReopenedBody     string // company | service, date, time
	apptCustCancelSubject, apptCustCancelBody string // company | service, date, time
	apptReschedSubject, apptReschedBody       string // company | service, date, time

	// Appointments — company
	coApptRequestedSubject  string // service, date
	coApptCustCancelSubject string // customer, date
	coApptReschedSubject    string // customer, date
	coApptIntro             string
	coApptCancelIntro       string
	coApptReschedIntro      string
	coApprovePending        string

	// Web orders — customer
	webCreatedSubject, webCreatedBody       string // ref, company | ref, company
	webApprovedSubject, webApprovedBody     string // ref | ref
	webRejectedSubject, webRejectedBody     string // ref | ref
	webCustCancelSubject, webCustCancelBody string // ref | ref

	// Web orders — company
	coWebCreatedSubject    string // ref, customer
	coWebCreatedIntro      string
	coWebCustCancelSubject string // ref
	coWebCustCancelIntro   string // ref, customer

	// Orders — customer
	orderCreatedSubject, orderCreatedBody string // ref, company | ref, company
	orderStatusSubject, orderStatusBody   string // ref, status | ref, status

	// More customer updates (2026-10-30)
	apptModifiedSubject, apptModifiedBody   string // company | service, date, time
	labelBefore                             string // previous date / time
	orderUpdatedSubject, orderUpdatedBody   string // ref | ref
	orderItemStepSubject, orderItemStepBody string // ref, product | product, step, step status
	orderStageSubject, orderStageBody       string // ref, stage | ref, stage, status

	// Cancellation requests (2026-10-23): the customer asks from the link,
	// the company cancels manually.
	cancelRequestLine  string // small line in customer emails
	kindAppointment    string
	kindWebOrder       string
	kindOrder          string
	cancelReqSubject   string // ref
	cancelReqBody      string // kind, ref, company
	coCancelReqSubject string // ref, customer
	coCancelReqIntro   string // kind
	labelRequestedAt   string
	labelCurrentStatus string

	// Payment (2026-10-22): instructions set in Company page + online
	// payments are not available yet.
	paymentTitle string
	paymentSoon  string

	statuses map[string]string

	// Company copy of a customer email (2026-10-30).
	copySubjectPrefix string
	copyIntro         string
	copySentTo        string // customer email
	copyNotSent       string // reason
	copyNoEmail       string

	// HTML emails / AlephK general email (2026-10-31).
	viaWord       string // sender name "Kire via AlephK"
	onBehalfLine  string // company name
	poweredByLine string
}

var textsEN = textSet{
	viaWord:           "via",
	onBehalfLine:      "Sent by AlephK on behalf of %s. To answer, just reply to this email.",
	poweredByLine:     "Powered by AlephK",
	copySubjectPrefix: "[Copy] ",
	copyIntro:         "Copy for the company of an email to your customer.",
	copySentTo:        "Sent to the customer: %s",
	copyNotSent:       "NOT sent to the customer: %s",
	copyNoEmail:       "the customer has no email",
	ourTeam:           "Our team",
	hi:                "Hi %s,",
	hiCompany:         "Hello,",
	thanks:            "Thank you,",
	whatsappLine:      "Questions? Talk to us on WhatsApp",
	manageLink:        "View or manage your request",
	viewOrder:         "See your order",
	allDay:            "all day",
	reasonLabel:       "Reason",
	paymentTitle:      "How to pay",
	paymentSoon:       "Online payments are coming soon. An agent will contact you with the payment instructions.",

	labelService: "Service", labelDate: "Date", labelTime: "Time", labelCustomer: "Customer",
	labelPhone: "Phone", labelEmail: "Email", labelComments: "Comments", labelReference: "Reference",
	labelStatus: "Status", labelTotal: "Total", labelItems: "Items", labelDelivery: "Delivery",
	labelAddress: "Address", labelNotes: "Notes",

	waAboutBooking: "Hello, I have a question about my booking (%s).",
	waAboutOrder:   "Hello, I have a question about my order %s.",

	apptRequestedSubject:  "We received your booking request — %s",
	apptRequestedBody:     "We received your request for %s on %s (%s).\n%s will review it and confirm it soon.",
	apptCreatedSubject:    "Your appointment is scheduled — %s",
	apptCreatedBody:       "Your appointment for %s is scheduled on %s (%s).",
	apptApprovedSubject:   "Your appointment is confirmed — %s",
	apptApprovedBody:      "Good news! Your appointment for %s on %s (%s) is confirmed.",
	apptRejectedSubject:   "About your booking request — %s",
	apptRejectedBody:      "We're sorry, we can't confirm your request for %s on %s (%s).\nYou are welcome to request another time.",
	apptDoneSubject:       "Thank you for your visit — %s",
	apptDoneBody:          "Your appointment for %s on %s was completed. We hope to see you again!",
	apptCancelledSubject:  "Your appointment was cancelled — %s",
	apptCancelledBody:     "Your appointment for %s on %s (%s) was cancelled.\nIf this is unexpected, please contact us.",
	apptReopenedSubject:   "Your appointment is scheduled again — %s",
	apptReopenedBody:      "Your appointment for %s on %s (%s) is scheduled again.",
	apptCustCancelSubject: "You cancelled your appointment — %s",
	apptCustCancelBody:    "As you asked, your appointment for %s on %s (%s) was cancelled.",
	apptReschedSubject:    "We received your new time request — %s",
	apptReschedBody:       "You asked to move your appointment for %s to %s (%s).\nThe new time must be confirmed; we will let you know.",

	coApptRequestedSubject:  "New booking request: %s, %s",
	coApptCustCancelSubject: "A customer cancelled: %s, %s",
	coApptReschedSubject:    "A customer asked for another time: %s, %s",
	coApptIntro:             "You have a new booking request.",
	coApptCancelIntro:       "A customer cancelled this appointment from their link.",
	coApptReschedIntro:      "A customer asked to move this appointment. It is PENDING again; approve it in Booking requests.",
	coApprovePending:        "Review it in Booking requests.",

	webCreatedSubject:    "We received your order %s — %s",
	webCreatedBody:       "We received your order request %s.\n%s will review it and contact you to confirm.",
	webApprovedSubject:   "Your order %s was accepted",
	webApprovedBody:      "Good news! Your order %s was accepted and is being prepared.",
	webRejectedSubject:   "About your order %s",
	webRejectedBody:      "We're sorry, we can't accept your order %s at this moment.\nPlease contact us if you have questions.",
	webCustCancelSubject: "You cancelled your order %s",
	webCustCancelBody:    "As you asked, your order request %s was cancelled.",

	coWebCreatedSubject:    "New web order %s — %s",
	coWebCreatedIntro:      "You have a new web order request. Review it in Web orders.",
	coWebCustCancelSubject: "A customer cancelled web order %s",
	coWebCustCancelIntro:   "Web order %s was cancelled by the customer (%s) from their link.",

	orderCreatedSubject: "Your order %s — %s",
	orderCreatedBody:    "Your order %s was registered by %s. We will keep you updated.",
	orderStatusSubject:  "Update on your order %s: %s",
	orderStatusBody:     "Your order %s has a new status: %s.",

	apptModifiedSubject:  "Your appointment was changed — %s",
	apptModifiedBody:     "Your appointment was changed. New details:\n%s, %s (%s).",
	labelBefore:          "Before",
	orderUpdatedSubject:  "Your order %s was updated",
	orderUpdatedBody:     "Your order %s was updated. These are the current details:",
	orderItemStepSubject: "Update on your order %s: %s",
	orderItemStepBody:    "%s is now in the step \"%s\" (%s).",
	orderStageSubject:    "Update on your order %s: %s",
	orderStageBody:       "Your order %s has a new update: %s (%s).",

	cancelRequestLine:  "Need to cancel? Request it here",
	kindAppointment:    "appointment",
	kindWebOrder:       "order request",
	kindOrder:          "order",
	cancelReqSubject:   "We received your cancellation request — %s",
	cancelReqBody:      "We received your request to cancel your %s %s.\n%s will review it and confirm the cancellation by email.",
	coCancelReqSubject: "Cancellation request: %s — %s",
	coCancelReqIntro:   "A customer asked to cancel their %s from their link.\nNothing was changed yet: please review it and cancel it manually in AlephTelos.",
	labelRequestedAt:   "Requested at",
	labelCurrentStatus: "Current status",

	statuses: map[string]string{
		"PENDING": "Pending", "APPROVED": "Approved", "REJECTED": "Rejected",
		"SCHEDULED": "Scheduled", "PLACED": "Placed", "IN_PROGRESS": "In progress",
		"PARTIAL_DONE": "Partially done", "DONE": "Done", "COMPLETED": "Completed",
		"READY": "Ready", "DELIVERED": "Delivered", "CANCELLED": "Cancelled",
		"ON_HOLD": "On hold", "SHIPPED": "Shipped",
	},
}

var textsES = textSet{
	viaWord:           "vía",
	onBehalfLine:      "Enviado por AlephK en nombre de %s. Para responder, simplemente responda a este correo.",
	poweredByLine:     "Con la tecnología de AlephK",
	copySubjectPrefix: "[Copia] ",
	copyIntro:         "Copia para la empresa de un correo a su cliente.",
	copySentTo:        "Enviado al cliente: %s",
	copyNotSent:       "NO se envió al cliente: %s",
	copyNoEmail:       "el cliente no tiene correo",
	ourTeam:           "Nuestro equipo",
	hi:                "Hola %s:",
	hiCompany:         "Hola:",
	thanks:            "Gracias,",
	whatsappLine:      "¿Preguntas? Escríbanos por WhatsApp",
	manageLink:        "Ver o gestionar su solicitud",
	viewOrder:         "Ver su pedido",
	allDay:            "todo el día",
	reasonLabel:       "Motivo",
	paymentTitle:      "Cómo pagar",
	paymentSoon:       "Los pagos en línea llegarán pronto. Un agente le contactará con las instrucciones de pago.",

	labelService: "Servicio", labelDate: "Fecha", labelTime: "Hora", labelCustomer: "Cliente",
	labelPhone: "Teléfono", labelEmail: "Correo", labelComments: "Comentarios", labelReference: "Referencia",
	labelStatus: "Estado", labelTotal: "Total", labelItems: "Artículos", labelDelivery: "Entrega",
	labelAddress: "Dirección", labelNotes: "Notas",

	waAboutBooking: "Hola, tengo una consulta sobre mi reserva (%s).",
	waAboutOrder:   "Hola, tengo una consulta sobre mi pedido %s.",

	apptRequestedSubject:  "Recibimos su solicitud de cita — %s",
	apptRequestedBody:     "Recibimos su solicitud para %s el %s (%s).\n%s la revisará y se la confirmará pronto.",
	apptCreatedSubject:    "Su cita está agendada — %s",
	apptCreatedBody:       "Su cita para %s está agendada el %s (%s).",
	apptApprovedSubject:   "Su cita está confirmada — %s",
	apptApprovedBody:      "¡Buenas noticias! Su cita para %s el %s (%s) está confirmada.",
	apptRejectedSubject:   "Sobre su solicitud de cita — %s",
	apptRejectedBody:      "Lo sentimos, no podemos confirmar su solicitud para %s el %s (%s).\nPuede solicitar otro horario cuando guste.",
	apptDoneSubject:       "Gracias por su visita — %s",
	apptDoneBody:          "Su cita para %s del %s fue completada. ¡Esperamos verle pronto!",
	apptCancelledSubject:  "Su cita fue cancelada — %s",
	apptCancelledBody:     "Su cita para %s el %s (%s) fue cancelada.\nSi no lo esperaba, por favor contáctenos.",
	apptReopenedSubject:   "Su cita fue agendada nuevamente — %s",
	apptReopenedBody:      "Su cita para %s el %s (%s) está agendada nuevamente.",
	apptCustCancelSubject: "Usted canceló su cita — %s",
	apptCustCancelBody:    "Como lo solicitó, su cita para %s el %s (%s) fue cancelada.",
	apptReschedSubject:    "Recibimos su solicitud de nuevo horario — %s",
	apptReschedBody:       "Usted pidió mover su cita para %s al %s (%s).\nEl nuevo horario debe ser confirmado; le avisaremos.",

	coApptRequestedSubject:  "Nueva solicitud de cita: %s, %s",
	coApptCustCancelSubject: "Un cliente canceló: %s, %s",
	coApptReschedSubject:    "Un cliente pidió otro horario: %s, %s",
	coApptIntro:             "Tiene una nueva solicitud de cita.",
	coApptCancelIntro:       "Un cliente canceló esta cita desde su enlace.",
	coApptReschedIntro:      "Un cliente pidió mover esta cita. Quedó PENDIENTE; apruébela en Solicitudes de reserva.",
	coApprovePending:        "Revísela en Solicitudes de reserva.",

	webCreatedSubject:    "Recibimos su pedido %s — %s",
	webCreatedBody:       "Recibimos su solicitud de pedido %s.\n%s la revisará y le contactará para confirmar.",
	webApprovedSubject:   "Su pedido %s fue aceptado",
	webApprovedBody:      "¡Buenas noticias! Su pedido %s fue aceptado y se está preparando.",
	webRejectedSubject:   "Sobre su pedido %s",
	webRejectedBody:      "Lo sentimos, en este momento no podemos aceptar su pedido %s.\nContáctenos si tiene preguntas.",
	webCustCancelSubject: "Usted canceló su pedido %s",
	webCustCancelBody:    "Como lo solicitó, su solicitud de pedido %s fue cancelada.",

	coWebCreatedSubject:    "Nuevo pedido web %s — %s",
	coWebCreatedIntro:      "Tiene una nueva solicitud de pedido web. Revísela en Pedidos web.",
	coWebCustCancelSubject: "Un cliente canceló el pedido web %s",
	coWebCustCancelIntro:   "El pedido web %s fue cancelado por el cliente (%s) desde su enlace.",

	orderCreatedSubject: "Su pedido %s — %s",
	orderCreatedBody:    "Su pedido %s fue registrado por %s. Le mantendremos informado.",
	orderStatusSubject:  "Actualización de su pedido %s: %s",
	orderStatusBody:     "Su pedido %s tiene un nuevo estado: %s.",

	apptModifiedSubject:  "Su cita fue modificada — %s",
	apptModifiedBody:     "Su cita fue modificada. Datos actuales:\n%s, %s (%s).",
	labelBefore:          "Antes",
	orderUpdatedSubject:  "Su pedido %s fue actualizado",
	orderUpdatedBody:     "Su pedido %s fue actualizado. Estos son los datos actuales:",
	orderItemStepSubject: "Actualización de su pedido %s: %s",
	orderItemStepBody:    "%s está ahora en el paso \"%s\" (%s).",
	orderStageSubject:    "Actualización de su pedido %s: %s",
	orderStageBody:       "Su pedido %s tiene una nueva actualización: %s (%s).",

	cancelRequestLine:  "¿Necesita cancelar? Solicítelo aquí",
	kindAppointment:    "cita",
	kindWebOrder:       "solicitud de pedido",
	kindOrder:          "pedido",
	cancelReqSubject:   "Recibimos su solicitud de cancelación — %s",
	cancelReqBody:      "Recibimos su solicitud para cancelar su %s %s.\n%s la revisará y le confirmará la cancelación por correo.",
	coCancelReqSubject: "Solicitud de cancelación: %s — %s",
	coCancelReqIntro:   "Un cliente solicitó cancelar su %s desde su enlace.\nTodavía no se cambió nada: revise la solicitud y cancele manualmente en AlephTelos.",
	labelRequestedAt:   "Solicitado el",
	labelCurrentStatus: "Estado actual",

	statuses: map[string]string{
		"PENDING": "Pendiente", "APPROVED": "Aprobado", "REJECTED": "Rechazado",
		"SCHEDULED": "Agendado", "PLACED": "Recibido", "IN_PROGRESS": "En proceso",
		"PARTIAL_DONE": "Parcialmente listo", "DONE": "Listo", "COMPLETED": "Completado",
		"READY": "Listo para entregar", "DELIVERED": "Entregado", "CANCELLED": "Cancelado",
		"ON_HOLD": "En espera", "SHIPPED": "Enviado",
	},
}
