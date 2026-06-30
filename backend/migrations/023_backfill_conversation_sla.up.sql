-- Backfill SLA data for open conversations that inherit a customer company from the contact.
UPDATE conversations conv
SET customer_company_id = ct.customer_company_id
FROM contacts ct
WHERE conv.contact_id = ct.id
  AND conv.customer_company_id IS NULL
  AND ct.customer_company_id IS NOT NULL;

UPDATE conversations conv
SET customer_company_id = COALESCE(conv.customer_company_id, ct.customer_company_id),
    first_response_due_at = COALESCE(
      conv.first_response_due_at,
      conv.created_at + (cc.initial_response_sla_minutes || ' minutes')::interval
    ),
    resolution_due_at = COALESCE(
      conv.resolution_due_at,
      conv.created_at + (cc.resolution_sla_minutes || ' minutes')::interval
    )
FROM contacts ct, customer_companies cc
WHERE conv.contact_id = ct.id
  AND COALESCE(conv.customer_company_id, ct.customer_company_id) = cc.id
  AND conv.status != 'resolved';
