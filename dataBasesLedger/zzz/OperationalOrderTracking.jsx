import { useMemo, useRef, useState } from "react";
import {
  FaArrowRotateLeft,
  FaChevronDown,
  FaCircleInfo,
  FaFloppyDisk,
  FaMagnifyingGlass,
  FaPaperclip,
  FaPlay,
  FaPlus,
  FaTrash,
  FaXmark,
} from "react-icons/fa6";
import "./OperationalOrderTracking.scss";

const STATUS_OPTIONS = [
  { value: "placed", labelKey: "operationalTracking.status.placed" },
  { value: "inReview", labelKey: "operationalTracking.status.inReview" },
  { value: "inProgress", labelKey: "operationalTracking.status.inProgress" },
  { value: "waitingCustomer", labelKey: "operationalTracking.status.waitingCustomer" },
  { value: "onHold", labelKey: "operationalTracking.status.onHold" },
  { value: "completed", labelKey: "operationalTracking.status.completed" },
];

const MOCK_ORDERS = [
  {
    id: "ORD-1001",
    orderName: "Custom family cutting board",
    customer: "Andrea Vargas",
    product: "Walnut cutting board",
    category: "Custom products",
    type: "product",
    orderDate: "2026-07-01",
    deliveryDate: "2026-07-08",
    followUpDate: "2026-07-05",
    role: "Carpenter",
    responsible: "Daniel Rojas",
    hasBooking: false,
    status: "placed",
    comments: [
      { id: 1, author: "Milbert Retana", date: "2026-07-01T09:20:00", text: "Customer has confirmed the logo and measurements." },
    ],
    attachments: [{ id: "a-1", name: "family-logo.png", size: 242000 }],
  },
  {
    id: "ORD-1002",
    orderName: "Vehicle diagnostic appointment",
    customer: "Carlos Jiménez",
    product: "Mechanical diagnostic",
    category: "Automotive services",
    type: "service",
    orderDate: "2026-07-02",
    deliveryDate: "2026-07-06",
    followUpDate: "2026-07-06",
    role: "Technician",
    responsible: "Laura Méndez",
    hasBooking: true,
    status: "inReview",
    comments: [],
    attachments: [],
  },
  {
    id: "ORD-1003",
    orderName: "Six engraved grill boards",
    customer: "Hotel Bahía Azul",
    product: "Grill board 30 × 50 cm",
    category: "Custom products",
    type: "product",
    orderDate: "2026-07-03",
    deliveryDate: "2026-07-12",
    followUpDate: "2026-07-08",
    role: "Engraver",
    responsible: "José Mora",
    hasBooking: false,
    status: "inProgress",
    comments: [],
    attachments: [{ id: "a-2", name: "hotel-brand.pdf", size: 416000 }],
  },
  {
    id: "ORD-1004",
    orderName: "Accounting consultation",
    customer: "María Fernández",
    product: "One-hour consultation",
    category: "Professional services",
    type: "service",
    orderDate: "2026-07-04",
    deliveryDate: "2026-07-10",
    followUpDate: "2026-07-09",
    role: "Consultant",
    responsible: "Sofía Castro",
    hasBooking: true,
    status: "waitingCustomer",
    comments: [],
    attachments: [],
  },
  {
    id: "ORD-1005",
    orderName: "Restaurant menu signs",
    customer: "Santo Pecado",
    product: "Laser-engraved signs",
    category: "Custom products",
    type: "product",
    orderDate: "2026-07-05",
    deliveryDate: "2026-07-15",
    followUpDate: "2026-07-11",
    role: "Designer",
    responsible: "Daniel Rojas",
    hasBooking: false,
    status: "onHold",
    comments: [],
    attachments: [],
  },
  {
    id: "ORD-1006",
    orderName: "Email configuration service",
    customer: "DC Group",
    product: "Microsoft 365 configuration",
    category: "Technical services",
    type: "service",
    orderDate: "2026-07-06",
    deliveryDate: "2026-07-07",
    followUpDate: "2026-07-08",
    role: "Technician",
    responsible: "Laura Méndez",
    hasBooking: true,
    status: "completed",
    comments: [],
    attachments: [],
  },
];

const EMPTY_FILTERS = {
  customer: "",
  startDate: "",
  endDate: "",
  category: "",
  type: "",
  orderName: "",
  product: "",
  booking: "",
  role: "",
  status: "",
};

const filterAndSortOrders = (orders, criteria) => {
  const includes = (value, term) => value.toLowerCase().includes(term.trim().toLowerCase());

  return [...orders]
    .filter((order) => !criteria.customer || order.customer === criteria.customer)
    .filter((order) => !criteria.startDate || order.orderDate >= criteria.startDate)
    .filter((order) => !criteria.endDate || order.orderDate <= criteria.endDate)
    .filter((order) => !criteria.category || order.category === criteria.category)
    .filter((order) => !criteria.type || order.type === criteria.type)
    .filter((order) => !criteria.orderName || includes(`${order.id} ${order.orderName}`, criteria.orderName))
    .filter((order) => !criteria.product || includes(order.product, criteria.product))
    .filter((order) => !criteria.role || order.role === criteria.role)
    .filter((order) => !criteria.status || order.status === criteria.status)
    .filter((order) => !criteria.booking || order.hasBooking === (criteria.booking === "yes"))
    .sort((a, b) => a.orderDate.localeCompare(b.orderDate) || a.id.localeCompare(b.id));
};

export default function OperationalOrderTracking({ t, locale = "en-US" }) {
  const [orders, setOrders] = useState(MOCK_ORDERS);
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [appliedFilters, setAppliedFilters] = useState(EMPTY_FILTERS);
  const [openStatusMenu, setOpenStatusMenu] = useState(null);
  const [selectedOrderId, setSelectedOrderId] = useState(null);
  const [detailStatus, setDetailStatus] = useState("placed");
  const [newComment, setNewComment] = useState("");
  const [message, setMessage] = useState("");
  const fileInputRef = useRef(null);

  const customers = useMemo(() => [...new Set(orders.map((order) => order.customer))].sort(), [orders]);
  const categories = useMemo(() => [...new Set(orders.map((order) => order.category))].sort(), [orders]);
  const roles = useMemo(() => [...new Set(orders.map((order) => order.role))].sort(), [orders]);

  const selectedOrder = orders.find((order) => order.id === selectedOrderId) || null;

  const visibleOrders = useMemo(
    () => filterAndSortOrders(orders, appliedFilters),
    [appliedFilters, orders],
  );

  const updateFilter = (event) => {
    const { name, value } = event.target;
    setFilters((current) => ({ ...current, [name]: value }));
  };

  const searchOrders = (event) => {
    event.preventDefault();
    setAppliedFilters(filters);
    const resultCount = filterAndSortOrders(orders, filters).length;
    setMessage(t("operationalTracking.messages.results", { count: resultCount }));
  };

  const clearFilters = () => {
    setFilters(EMPTY_FILTERS);
    setAppliedFilters(EMPTY_FILTERS);
    setMessage("");
  };

  const changeStatus = (orderId, status) => {
    setOrders((current) => current.map((order) => (
      order.id === orderId ? { ...order, status } : order
    )));
    setOpenStatusMenu(null);
    setMessage(t("operationalTracking.messages.statusChanged"));
  };

  const openDetails = (order) => {
    setSelectedOrderId(order.id);
    setDetailStatus(order.status);
    setNewComment("");
    setOpenStatusMenu(null);
  };

  const closeDetails = () => {
    setSelectedOrderId(null);
    setNewComment("");
  };

  const addComment = () => {
    const text = newComment.trim();
    if (!text || !selectedOrderId) return;

    setOrders((current) => current.map((order) => order.id === selectedOrderId ? {
      ...order,
      comments: [
        ...order.comments,
        { id: Date.now(), author: "Current user", date: new Date().toISOString(), text },
      ],
    } : order));
    setNewComment("");
  };

  const addFiles = (event) => {
    const files = Array.from(event.target.files || []);
    if (!files.length || !selectedOrderId) return;

    setOrders((current) => current.map((order) => order.id === selectedOrderId ? {
      ...order,
      attachments: [
        ...order.attachments,
        ...files.map((file) => ({ id: `${file.name}-${file.lastModified}`, name: file.name, size: file.size, file })),
      ],
    } : order));
    event.target.value = "";
  };

  const removeAttachment = (attachmentId) => {
    setOrders((current) => current.map((order) => order.id === selectedOrderId ? {
      ...order,
      attachments: order.attachments.filter((attachment) => attachment.id !== attachmentId),
    } : order));
  };

  const saveDetails = () => {
    if (!selectedOrderId) return;
    setOrders((current) => current.map((order) => (
      order.id === selectedOrderId ? { ...order, status: detailStatus } : order
    )));
    setMessage(t("operationalTracking.messages.detailsSaved"));
    closeDetails();
  };

  const formatDate = (value) => new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "short",
    day: "2-digit",
  }).format(new Date(`${value}T12:00:00`));

  const formatDateTime = (value) => new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));

  return (
    <section className="operational-tracking">
      <header className="operational-tracking__header">
        <div>
          <p className="operational-tracking__eyebrow">{t("operationalTracking.eyebrow")}</p>
          <h1>{t("operationalTracking.title")}</h1>
          <p>{t("operationalTracking.description")}</p>
        </div>
        <span className="operational-tracking__count">
          {t("operationalTracking.orderCount", { count: visibleOrders.length })}
        </span>
      </header>

      <form className="tracking-filters" onSubmit={searchOrders}>
        <div className="tracking-filters__title">
          <div><FaMagnifyingGlass /><span>{t("operationalTracking.filters.title")}</span></div>
          <small>{t("operationalTracking.filters.help")}</small>
        </div>

        <div className="tracking-filters__grid">
          <label><span>{t("operationalTracking.filters.customer")}</span><select name="customer" value={filters.customer} onChange={updateFilter}><option value="">{t("operationalTracking.common.all")}</option>{customers.map((customer) => <option key={customer}>{customer}</option>)}</select></label>
          <label><span>{t("operationalTracking.filters.startDate")}</span><input name="startDate" type="date" value={filters.startDate} onChange={updateFilter} /></label>
          <label><span>{t("operationalTracking.filters.endDate")}</span><input name="endDate" type="date" value={filters.endDate} onChange={updateFilter} /></label>
          <label><span>{t("operationalTracking.filters.category")}</span><select name="category" value={filters.category} onChange={updateFilter}><option value="">{t("operationalTracking.common.all")}</option>{categories.map((category) => <option key={category}>{category}</option>)}</select></label>
          <label><span>{t("operationalTracking.filters.type")}</span><select name="type" value={filters.type} onChange={updateFilter}><option value="">{t("operationalTracking.common.all")}</option><option value="product">{t("operationalTracking.type.product")}</option><option value="service">{t("operationalTracking.type.service")}</option></select></label>
          <label><span>{t("operationalTracking.filters.orderName")}</span><input name="orderName" value={filters.orderName} onChange={updateFilter} placeholder={t("operationalTracking.filters.orderPlaceholder")} /></label>
          <label><span>{t("operationalTracking.filters.product")}</span><input name="product" value={filters.product} onChange={updateFilter} placeholder={t("operationalTracking.filters.productPlaceholder")} /></label>
          <label><span>{t("operationalTracking.filters.booking")}</span><select name="booking" value={filters.booking} onChange={updateFilter}><option value="">{t("operationalTracking.common.all")}</option><option value="yes">{t("operationalTracking.common.yes")}</option><option value="no">{t("operationalTracking.common.no")}</option></select></label>
          <label><span>{t("operationalTracking.filters.role")}</span><select name="role" value={filters.role} onChange={updateFilter}><option value="">{t("operationalTracking.common.all")}</option>{roles.map((role) => <option key={role}>{role}</option>)}</select></label>
          <label><span>{t("operationalTracking.filters.status")}</span><select name="status" value={filters.status} onChange={updateFilter}><option value="">{t("operationalTracking.common.all")}</option>{STATUS_OPTIONS.map((status) => <option value={status.value} key={status.value}>{t(status.labelKey)}</option>)}</select></label>
        </div>

        <div className="tracking-filters__actions">
          <button className="tracking-button tracking-button--ghost" type="button" onClick={clearFilters}><FaArrowRotateLeft />{t("operationalTracking.actions.clear")}</button>
          <button className="tracking-button tracking-button--primary" type="submit"><FaMagnifyingGlass />{t("operationalTracking.actions.search")}</button>
        </div>
      </form>

      <div className="order-list-section">
        <div className="order-list-section__title">
          <div><h2>{t("operationalTracking.list.title")}</h2><p>{t("operationalTracking.list.oldestFirst")}</p></div>
          <p role="status">{message}</p>
        </div>

        <div className="order-table-wrap">
          <table className="order-table">
            <thead><tr>
              <th>{t("operationalTracking.columns.number")}</th>
              <th>{t("operationalTracking.columns.order")}</th>
              <th>{t("operationalTracking.columns.customer")}</th>
              <th>{t("operationalTracking.columns.product")}</th>
              <th>{t("operationalTracking.columns.type")}</th>
              <th>{t("operationalTracking.columns.date")}</th>
              <th>{t("operationalTracking.columns.status")}</th>
              <th>{t("operationalTracking.columns.actions")}</th>
            </tr></thead>
            <tbody>
              {visibleOrders.map((order) => (
                <tr key={order.id}>
                  <td><strong>{order.id}</strong></td>
                  <td><strong>{order.orderName}</strong><small>{order.responsible} · {order.role}</small></td>
                  <td>{order.customer}</td>
                  <td><span>{order.product}</span><small>{order.category}</small></td>
                  <td><span className={`order-type order-type--${order.type}`}>{t(`operationalTracking.type.${order.type}`)}</span>{order.hasBooking && <small>{t("operationalTracking.list.booking")}</small>}</td>
                  <td><span>{formatDate(order.orderDate)}</span><small>{t("operationalTracking.list.delivery")}: {formatDate(order.deliveryDate)}</small></td>
                  <td><span className={`order-status order-status--${order.status}`}>{t(`operationalTracking.status.${order.status}`)}</span></td>
                  <td>
                    <div className="order-actions">
                      <button type="button" className="tracking-button tracking-button--detail" onClick={() => openDetails(order)}><FaCircleInfo />{t("operationalTracking.actions.details")}</button>
                      <div className="status-menu">
                        <button type="button" className="tracking-button tracking-button--status" onClick={() => setOpenStatusMenu(openStatusMenu === order.id ? null : order.id)}><FaPlay />{t("operationalTracking.actions.changeStatus")}<FaChevronDown /></button>
                        {openStatusMenu === order.id && <div className="status-menu__options">{STATUS_OPTIONS.map((status) => <button type="button" key={status.value} onClick={() => changeStatus(order.id, status.value)}><span className={`status-dot status-dot--${status.value}`} />{t(status.labelKey)}</button>)}</div>}
                      </div>
                    </div>
                  </td>
                </tr>
              ))}
              {visibleOrders.length === 0 && <tr><td className="order-table__empty" colSpan="8">{t("operationalTracking.list.empty")}</td></tr>}
            </tbody>
          </table>
        </div>
      </div>

      {selectedOrder && (
        <div className="tracking-modal" role="dialog" aria-modal="true" aria-labelledby="tracking-detail-title">
          <button className="tracking-modal__backdrop" type="button" onClick={closeDetails} aria-label={t("operationalTracking.actions.close")} />
          <aside className="tracking-detail">
            <header className="tracking-detail__header">
              <div><p>{selectedOrder.id}</p><h2 id="tracking-detail-title">{selectedOrder.orderName}</h2><span>{selectedOrder.customer}</span></div>
              <button type="button" onClick={closeDetails} aria-label={t("operationalTracking.actions.close")}><FaXmark /></button>
            </header>

            <div className="tracking-detail__body">
              <div className="tracking-detail__summary">
                <div><span>{t("operationalTracking.detail.responsible")}</span><strong>{selectedOrder.responsible}</strong></div>
                <div><span>{t("operationalTracking.detail.delivery")}</span><strong>{formatDate(selectedOrder.deliveryDate)}</strong></div>
                <div><span>{t("operationalTracking.detail.followUp")}</span><strong>{formatDate(selectedOrder.followUpDate)}</strong></div>
              </div>

              <label className="tracking-detail__status"><span>{t("operationalTracking.detail.status")}</span><select value={detailStatus} onChange={(event) => setDetailStatus(event.target.value)}>{STATUS_OPTIONS.map((status) => <option value={status.value} key={status.value}>{t(status.labelKey)}</option>)}</select></label>

              <section className="tracking-detail__section">
                <h3>{t("operationalTracking.detail.comments")}</h3>
                <div className="comment-list">
                  {selectedOrder.comments.length === 0 && <p>{t("operationalTracking.detail.noComments")}</p>}
                  {selectedOrder.comments.map((comment) => <article key={comment.id}><div><strong>{comment.author}</strong><small>{formatDateTime(comment.date)}</small></div><p>{comment.text}</p></article>)}
                </div>
                <textarea rows="3" value={newComment} onChange={(event) => setNewComment(event.target.value)} placeholder={t("operationalTracking.detail.commentPlaceholder")} />
                <button className="tracking-button tracking-button--ghost" type="button" onClick={addComment}><FaPlus />{t("operationalTracking.actions.addComment")}</button>
              </section>

              <section className="tracking-detail__section">
                <h3>{t("operationalTracking.detail.attachments")}</h3>
                <input ref={fileInputRef} type="file" multiple onChange={addFiles} />
                <button className="attachment-drop" type="button" onClick={() => fileInputRef.current?.click()}><FaPaperclip /><span>{t("operationalTracking.actions.addFiles")}</span></button>
                <div className="detail-files">{selectedOrder.attachments.map((attachment) => <div key={attachment.id}><span><strong>{attachment.name}</strong><small>{Math.round(attachment.size / 1024)} KB</small></span><button type="button" onClick={() => removeAttachment(attachment.id)} aria-label={t("operationalTracking.actions.removeFile")}><FaTrash /></button></div>)}</div>
              </section>
            </div>

            <footer className="tracking-detail__footer">
              <button className="tracking-button tracking-button--ghost" type="button" onClick={closeDetails}>{t("operationalTracking.actions.close")}</button>
              <button className="tracking-button tracking-button--primary" type="button" onClick={saveDetails}><FaFloppyDisk />{t("operationalTracking.actions.save")}</button>
            </footer>
          </aside>
        </div>
      )}
    </section>
  );
}