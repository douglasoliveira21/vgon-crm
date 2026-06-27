package handlers

import (
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type sipTrunkBody struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SIPServer       string   `json:"sip_server"`
	SIPPort         int      `json:"sip_port"`
	Transport       string   `json:"transport"`
	SIPDomain       string   `json:"sip_domain"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	CallerID        string   `json:"caller_id"`
	Realm           string   `json:"realm"`
	OutboundProxy   string   `json:"outbound_proxy"`
	Codecs          []string `json:"codecs"`
	NAT             bool     `json:"nat"`
	KeepAlive       int      `json:"keep_alive"`
	DTMF            string   `json:"dtmf"`
	RegisterExpires int      `json:"register_expires"`
	IsActive        bool     `json:"is_active"`
}

func GetSIPTrunks(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query(`
			SELECT id, name, COALESCE(description, ''), sip_server, sip_port, transport,
			       COALESCE(sip_domain, ''), username, caller_id, COALESCE(realm, ''),
			       COALESCE(outbound_proxy, ''), codecs, nat, keep_alive, dtmf,
			       register_expires, is_active, created_at
			FROM sip_trunks
			WHERE company_id = $1
			ORDER BY name
		`, companyID)
		if err != nil {
			return c.JSON(fiber.Map{"trunks": []interface{}{}})
		}
		defer rows.Close()

		trunks := []map[string]interface{}{}
		for rows.Next() {
			var id, name, description, server, transport, domain, username, callerID, realm, proxy, dtmf string
			var codecs []byte
			var port, keepAlive, registerExpires int
			var nat, active bool
			var createdAt interface{}
			rows.Scan(&id, &name, &description, &server, &port, &transport, &domain, &username, &callerID, &realm, &proxy, &codecs, &nat, &keepAlive, &dtmf, &registerExpires, &active, &createdAt)
			trunks = append(trunks, map[string]interface{}{
				"id": id, "name": name, "description": description, "sip_server": server,
				"sip_port": port, "transport": transport, "sip_domain": domain,
				"username": username, "caller_id": callerID, "realm": realm,
				"outbound_proxy": proxy, "codecs": string(codecs), "nat": nat,
				"keep_alive": keepAlive, "dtmf": dtmf, "register_expires": registerExpires,
				"is_active": active, "created_at": createdAt,
			})
		}
		return c.JSON(fiber.Map{"trunks": trunks})
	}
}

func CreateSIPTrunk(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return saveSIPTrunk(c, svc, "")
	}
}

func UpdateSIPTrunk(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return saveSIPTrunk(c, svc, c.Params("id"))
	}
}

func DeleteSIPTrunk(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")
		_, err := svc.DB.Exec("DELETE FROM sip_trunks WHERE id = $1 AND company_id = $2", id, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Trunk deleted"})
	}
}

func saveSIPTrunk(c *fiber.Ctx, svc *services.Container, id string) error {
	companyID := c.Locals("company_id").(string)
	var body sipTrunkBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}
	if body.Name == "" || body.SIPServer == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and SIP server are required"})
	}
	if body.SIPPort == 0 {
		body.SIPPort = 5060
	}
	if body.Transport == "" {
		body.Transport = "UDP"
	}
	if body.DTMF == "" {
		body.DTMF = "rfc4733"
	}
	if body.KeepAlive == 0 {
		body.KeepAlive = 60
	}
	if body.RegisterExpires == 0 {
		body.RegisterExpires = 300
	}
	if len(body.Codecs) == 0 {
		body.Codecs = []string{"ulaw", "alaw"}
	}
	configPassword := body.Password
	encryptedPassword, err := services.EncryptSecret(body.Password, svc.Config.JWTSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to encrypt trunk password"})
	}
	if id != "" && body.Password == "" {
		var currentPassword string
		_ = svc.DB.QueryRow("SELECT password FROM sip_trunks WHERE id = $1 AND company_id = $2", id, companyID).Scan(&currentPassword)
		encryptedPassword = currentPassword
		configPassword = services.DecryptSecret(currentPassword, svc.Config.JWTSecret)
	}

	pjsipConfig := svc.Asterisk.GeneratePJSIPTrunkConfig(id, body.Name, body.SIPServer, body.SIPPort, body.Transport, body.SIPDomain, body.Username, configPassword, body.CallerID, body.Realm, body.OutboundProxy, body.Codecs, body.NAT, body.KeepAlive, body.DTMF, body.RegisterExpires)
	codecsJSON := services.StringsJSON(body.Codecs)

	if id == "" {
		id = uuid.New().String()
		pjsipConfig = svc.Asterisk.GeneratePJSIPTrunkConfig(id, body.Name, body.SIPServer, body.SIPPort, body.Transport, body.SIPDomain, body.Username, configPassword, body.CallerID, body.Realm, body.OutboundProxy, body.Codecs, body.NAT, body.KeepAlive, body.DTMF, body.RegisterExpires)
		_, err = svc.DB.Exec(`
			INSERT INTO sip_trunks (id, company_id, name, description, sip_server, sip_port, transport, sip_domain,
				username, password, caller_id, realm, outbound_proxy, codecs, nat, keep_alive, dtmf, register_expires, pjsip_config, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15, $16, $17, $18, $19, $20)
		`, id, companyID, body.Name, body.Description, body.SIPServer, body.SIPPort, body.Transport, body.SIPDomain, body.Username, encryptedPassword, body.CallerID, body.Realm, body.OutboundProxy, codecsJSON, body.NAT, body.KeepAlive, body.DTMF, body.RegisterExpires, pjsipConfig, body.IsActive)
	} else {
		_, err = svc.DB.Exec(`
			UPDATE sip_trunks
			SET name=$1, description=$2, sip_server=$3, sip_port=$4, transport=$5, sip_domain=$6,
			    username=$7, password=$8, caller_id=$9, realm=$10, outbound_proxy=$11, codecs=$12::jsonb,
			    nat=$13, keep_alive=$14, dtmf=$15, register_expires=$16, pjsip_config=$17, is_active=$18, updated_at=NOW()
			WHERE id=$19 AND company_id=$20
		`, body.Name, body.Description, body.SIPServer, body.SIPPort, body.Transport, body.SIPDomain, body.Username, encryptedPassword, body.CallerID, body.Realm, body.OutboundProxy, codecsJSON, body.NAT, body.KeepAlive, body.DTMF, body.RegisterExpires, pjsipConfig, body.IsActive, id, companyID)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	_ = svc.Asterisk.ReloadPJSIP(companyID)
	return c.JSON(fiber.Map{"id": id, "pjsip_config": pjsipConfig})
}

func SaveTelephonyProvider(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body telephonyProviderBody
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		setTelephonyDefaults(&body)

		sipPassword, err := services.EncryptSecret(body.SipPassword, svc.Config.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to encrypt SIP password"})
		}
		ariPassword, err := services.EncryptSecret(body.ARIPassword, svc.Config.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to encrypt ARI password"})
		}
		amiPassword, err := services.EncryptSecret(body.AMIPassword, svc.Config.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to encrypt AMI password"})
		}

		// Check if provider already exists for this company
		var existingID string
		err = svc.DB.QueryRow("SELECT id FROM telephony_providers WHERE company_id = $1 LIMIT 1", companyID).Scan(&existingID)

		if err == nil {
			// Update existing
			_, err := svc.DB.Exec(`
				UPDATE telephony_providers
				SET name=$1, provider_type=$2, sip_host=$3, sip_port=$4, sip_user=$5, sip_password=$6,
				    sip_domain=$7, webrtc_domain=$8, webrtc_ws_url=$9, transport=$10, caller_id=$11,
				    stun_server=$12, ari_url=$13, ari_user=$14, ari_password=$15, ami_host=$16,
				    ami_port=$17, ami_user=$18, ami_password=$19, recording_path=$20,
				    recording_enabled=$21, updated_at=NOW()
				WHERE id=$22
			`, body.Name, body.ProviderType, body.SipHost, body.SipPort, body.SipUser, sipPassword, body.SipDomain,
				body.WebRTCDomain, body.WebRTCWSURL, body.Transport, body.CallerID, body.StunServer,
				body.ARIURL, body.ARIUser, ariPassword, body.AMIHost, body.AMIPort, body.AMIUser,
				amiPassword, body.RecordingPath, body.RecordingEnabled, existingID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			return c.JSON(fiber.Map{"message": "Provider updated", "id": existingID})
		}

		// Create new
		id := uuid.New().String()
		_, err = svc.DB.Exec(`
			INSERT INTO telephony_providers (
				id, company_id, name, provider_type, sip_host, sip_port, sip_user, sip_password,
				sip_domain, webrtc_domain, webrtc_ws_url, transport, caller_id, stun_server,
				ari_url, ari_user, ari_password, ami_host, ami_port, ami_user, ami_password,
				recording_path, recording_enabled
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		`, id, companyID, body.Name, body.ProviderType, body.SipHost, body.SipPort, body.SipUser,
			sipPassword, body.SipDomain, body.WebRTCDomain, body.WebRTCWSURL, body.Transport,
			body.CallerID, body.StunServer, body.ARIURL, body.ARIUser, ariPassword, body.AMIHost,
			body.AMIPort, body.AMIUser, amiPassword, body.RecordingPath, body.RecordingEnabled)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Provider created", "id": id})
	}
}

func GetTelephonyProvider(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var id, name, providerType, sipHost, sipUser, sipDomain, webRTCDomain, webRTCWSURL, transport, callerID, stunServer, status string
		var ariURL, ariUser, amiHost, amiUser, recordingPath string
		var sipPort, amiPort int
		var recordingEnabled bool

		err := svc.DB.QueryRow(`
			SELECT id, name, provider_type, sip_host, sip_port, sip_user, sip_domain,
			       COALESCE(webrtc_domain, sip_domain, sip_host), COALESCE(webrtc_ws_url, ''),
			       transport, caller_id, COALESCE(stun_server, ''), COALESCE(ari_url, ''),
			       COALESCE(ari_user, ''), COALESCE(ami_host, ''), COALESCE(ami_port, 5038),
			       COALESCE(ami_user, ''), COALESCE(recording_path, '/var/spool/asterisk/monitor'),
			       recording_enabled, status
			FROM telephony_providers WHERE company_id = $1 LIMIT 1
		`, companyID).Scan(&id, &name, &providerType, &sipHost, &sipPort, &sipUser, &sipDomain,
			&webRTCDomain, &webRTCWSURL, &transport, &callerID, &stunServer, &ariURL, &ariUser,
			&amiHost, &amiPort, &amiUser, &recordingPath, &recordingEnabled, &status)

		if err != nil {
			return c.JSON(fiber.Map{"provider": nil})
		}

		return c.JSON(fiber.Map{"provider": fiber.Map{
			"id": id, "name": name, "provider_type": providerType,
			"sip_host": sipHost, "sip_port": sipPort, "sip_user": sipUser,
			"sip_domain": sipDomain, "webrtc_domain": webRTCDomain, "webrtc_ws_url": webRTCWSURL,
			"transport": transport, "caller_id": callerID, "stun_server": stunServer,
			"ari_url": ariURL, "ari_user": ariUser, "ami_host": amiHost, "ami_port": amiPort,
			"ami_user": amiUser, "recording_path": recordingPath,
			"recording_enabled": recordingEnabled, "status": status,
		}})
	}
}

func GetExtensions(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query(`
			SELECT pe.id, COALESCE(pe.user_id::text, ''), pe.extension_number, pe.display_name, pe.status,
			       pe.can_call_external, pe.can_receive_calls, pe.can_transfer, pe.can_access_recordings,
			       COALESCE(pe.outbound_trunk_id::text, ''), COALESCE(st.name, ''),
			       COALESCE(pe.webrtc_domain, ''), COALESCE(pe.webrtc_ws_url, ''), COALESCE(pe.stun_server, ''),
			       COALESCE(pe.sip_username, pe.extension_number), COALESCE(pe.group_name, ''), COALESCE(pe.queue_id::text, '')
			FROM phone_extensions pe
			LEFT JOIN sip_trunks st ON st.id = pe.outbound_trunk_id
			WHERE pe.company_id = $1 ORDER BY pe.extension_number
		`, companyID)
		if err != nil {
			return c.JSON(fiber.Map{"extensions": []interface{}{}})
		}
		defer rows.Close()

		var extensions []map[string]interface{}
		for rows.Next() {
			var id, userID, number, name, status, trunkID, trunkName, webrtcDomain, webrtcWSURL, stunServer, sipUsername, groupName, queueID string
			var canCallExt, canReceive, canTransfer, canAccessRec bool
			rows.Scan(&id, &userID, &number, &name, &status, &canCallExt, &canReceive, &canTransfer, &canAccessRec, &trunkID, &trunkName, &webrtcDomain, &webrtcWSURL, &stunServer, &sipUsername, &groupName, &queueID)
			extensions = append(extensions, map[string]interface{}{
				"id": id, "user_id": userID, "extension_number": number, "display_name": name,
				"status": status, "can_call_external": canCallExt, "can_receive_calls": canReceive,
				"can_transfer": canTransfer, "can_access_recordings": canAccessRec,
				"outbound_trunk_id": trunkID, "outbound_trunk_name": trunkName,
				"webrtc_domain": webrtcDomain, "webrtc_ws_url": webrtcWSURL,
				"stun_server": stunServer, "sip_username": sipUsername,
				"group_name": groupName, "queue_id": queueID,
			})
		}
		return c.JSON(fiber.Map{"extensions": extensions})
	}
}

func CreateExtension(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		var body struct {
			UserID            string `json:"user_id"`
			QueueID           string `json:"queue_id"`
			GroupName         string `json:"group_name"`
			OutboundTrunkID   string `json:"outbound_trunk_id"`
			DisplayName       string `json:"display_name"`
			ExtensionNumber   string `json:"extension_number"`
			ExtensionPassword string `json:"extension_password"`
			SIPUsername       string `json:"sip_username"`
			WebRTCDomain      string `json:"webrtc_domain"`
			WebRTCWSURL       string `json:"webrtc_ws_url"`
			StunServer        string `json:"stun_server"`
			CanCallExternal   bool   `json:"can_call_external"`
			CanReceiveCalls   bool   `json:"can_receive_calls"`
			CanTransfer       bool   `json:"can_transfer"`
			CanAccessRec      bool   `json:"can_access_recordings"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.ExtensionNumber == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Extension number is required"})
		}
		if body.ExtensionPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Extension password is required"})
		}
		if body.SIPUsername == "" {
			body.SIPUsername = body.ExtensionNumber
		}
		if body.WebRTCDomain == "" {
			body.WebRTCDomain = "voip.vgon.com.br"
		}
		if body.WebRTCWSURL == "" {
			body.WebRTCWSURL = "wss://voip.vgon.com.br:8089/ws"
		}
		if body.StunServer == "" {
			body.StunServer = "stun:stun.l.google.com:19302"
		}

		extensionPassword, err := services.EncryptSecret(body.ExtensionPassword, svc.Config.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to encrypt extension password"})
		}

		id := ""
		err = svc.DB.QueryRow(`
			SELECT id
			FROM phone_extensions
			WHERE company_id = $1 AND extension_number = $2
			ORDER BY created_at
			LIMIT 1
		`, companyID, body.ExtensionNumber).Scan(&id)
		if err == nil {
			_, err = svc.DB.Exec(`
				UPDATE phone_extensions
				SET user_id = NULLIF($1, '')::uuid,
					queue_id = NULLIF($2, '')::uuid,
					group_name = $3,
					outbound_trunk_id = NULLIF($4, '')::uuid,
					display_name = $5,
					extension_password = $6,
					sip_username = $7,
					webrtc_domain = $8,
					webrtc_ws_url = $9,
					stun_server = $10,
					can_call_external = $11,
					can_receive_calls = $12,
					can_transfer = $13,
					can_access_recordings = $14,
					updated_at = NOW()
				WHERE id = $15 AND company_id = $16
			`, body.UserID, body.QueueID, body.GroupName, body.OutboundTrunkID, body.DisplayName, extensionPassword, body.SIPUsername, body.WebRTCDomain, body.WebRTCWSURL, body.StunServer, body.CanCallExternal, body.CanReceiveCalls, body.CanTransfer, body.CanAccessRec, id, companyID)
		} else {
			id = uuid.New().String()
			_, err = svc.DB.Exec(`
				INSERT INTO phone_extensions (
					id, company_id, user_id, queue_id, group_name, outbound_trunk_id, display_name,
					extension_number, extension_password, sip_username, webrtc_domain, webrtc_ws_url,
					stun_server, can_call_external, can_receive_calls, can_transfer, can_access_recordings
				)
				VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5, NULLIF($6, '')::uuid, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			`, id, companyID, body.UserID, body.QueueID, body.GroupName, body.OutboundTrunkID, body.DisplayName, body.ExtensionNumber, extensionPassword, body.SIPUsername, body.WebRTCDomain, body.WebRTCWSURL, body.StunServer, body.CanCallExternal, body.CanReceiveCalls, body.CanTransfer, body.CanAccessRec)
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

type telephonyProviderBody struct {
	Name             string `json:"name"`
	ProviderType     string `json:"provider_type"`
	SipHost          string `json:"sip_host"`
	SipPort          int    `json:"sip_port"`
	SipUser          string `json:"sip_user"`
	SipPassword      string `json:"sip_password"`
	SipDomain        string `json:"sip_domain"`
	WebRTCDomain     string `json:"webrtc_domain"`
	WebRTCWSURL      string `json:"webrtc_ws_url"`
	Transport        string `json:"transport"`
	CallerID         string `json:"caller_id"`
	StunServer       string `json:"stun_server"`
	ARIURL           string `json:"ari_url"`
	ARIUser          string `json:"ari_user"`
	ARIPassword      string `json:"ari_password"`
	AMIHost          string `json:"ami_host"`
	AMIPort          int    `json:"ami_port"`
	AMIUser          string `json:"ami_user"`
	AMIPassword      string `json:"ami_password"`
	RecordingPath    string `json:"recording_path"`
	RecordingEnabled bool   `json:"recording_enabled"`
}

func setTelephonyDefaults(body *telephonyProviderBody) {
	if body.Name == "" {
		body.Name = "Asterisk VGoN"
	}
	if body.ProviderType == "" {
		body.ProviderType = "asterisk"
	}
	if body.SipHost == "" {
		body.SipHost = "voip.vgon.com.br"
	}
	if body.SipPort == 0 {
		body.SipPort = 5060
	}
	if body.SipDomain == "" {
		body.SipDomain = "voip.vgon.com.br"
	}
	if body.WebRTCDomain == "" {
		body.WebRTCDomain = "voip.vgon.com.br"
	}
	if body.WebRTCWSURL == "" {
		body.WebRTCWSURL = "wss://voip.vgon.com.br:8089/ws"
	}
	if body.Transport == "" {
		body.Transport = "WSS"
	}
	if body.StunServer == "" {
		body.StunServer = "stun:stun.l.google.com:19302"
	}
	if body.ARIURL == "" {
		body.ARIURL = "http://voip.vgon.com.br:8088/ari"
	}
	if body.AMIHost == "" {
		body.AMIHost = "85.239.248.224"
	}
	if body.AMIPort == 0 {
		body.AMIPort = 5038
	}
	if body.RecordingPath == "" {
		body.RecordingPath = "/var/spool/asterisk/monitor"
	}
}

func DeleteExtension(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")
		svc.DB.Exec("DELETE FROM phone_extensions WHERE id = $1 AND company_id = $2", id, companyID)
		return c.JSON(fiber.Map{"message": "Extension deleted"})
	}
}

func GetQueues(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query("SELECT id, name, strategy FROM call_queues WHERE company_id = $1 ORDER BY name", companyID)
		if err != nil {
			return c.JSON(fiber.Map{"queues": []interface{}{}})
		}
		defer rows.Close()

		var queues []map[string]interface{}
		for rows.Next() {
			var id, name, strategy string
			rows.Scan(&id, &name, &strategy)
			queues = append(queues, map[string]interface{}{"id": id, "name": name, "strategy": strategy})
		}
		return c.JSON(fiber.Map{"queues": queues})
	}
}

func CreateQueue(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		var body struct {
			Name        string `json:"name"`
			Strategy    string `json:"strategy"`
			MaxWaitTime int    `json:"max_wait_time"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.Strategy == "" {
			body.Strategy = "ringall"
		}
		if body.MaxWaitTime == 0 {
			body.MaxWaitTime = 120
		}

		id := uuid.New().String()
		svc.DB.Exec("INSERT INTO call_queues (id, company_id, name, strategy, max_wait_time) VALUES ($1, $2, $3, $4, $5)",
			id, companyID, body.Name, body.Strategy, body.MaxWaitTime)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func DeleteQueue(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")
		svc.DB.Exec("DELETE FROM call_queues WHERE id = $1 AND company_id = $2", id, companyID)
		return c.JSON(fiber.Map{"message": "Queue deleted"})
	}
}
